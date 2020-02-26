package integration

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/Fantom-foundation/go-lachesis/app"
	"github.com/Fantom-foundation/go-lachesis/gossip"
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/kvdb/flushable"
	"github.com/Fantom-foundation/go-lachesis/poset"
)

// MakeEngine makes consensus engine from config.
func MakeEngine(dataDir string, gossipCfg *gossip.Config) (*poset.Poset, *app.Store, *gossip.Store) {
	dbs := flushable.NewSyncedPool(dbProducer(dataDir))

	appStoreConfig := app.StoreConfig{
		ReceiptsCacheSize:   gossipCfg.ReceiptsCacheSize,
		DelegatorsCacheSize: gossipCfg.DelegatorsCacheSize,
		StakersCacheSize:    gossipCfg.StakersCacheSize,
	}
	adb := app.NewStore(dbs, appStoreConfig)
	gdb := gossip.NewStore(dbs, gossipCfg.StoreConfig)
	cdb := poset.NewStore(dbs, poset.DefaultStoreConfig())

	// write genesis

	state, _, err := adb.ApplyGenesis(&gossipCfg.Net)
	if err != nil {
		utils.Fatalf("Failed to write App genesis state: %v", err)
	}

	genesisAtropos, genesisState, isNew, err := gdb.ApplyGenesis(&gossipCfg.Net, state)
	if err != nil {
		utils.Fatalf("Failed to write Gossip genesis state: %v", err)
	}

	err = cdb.ApplyGenesis(&gossipCfg.Net.Genesis, genesisAtropos, genesisState)
	if err != nil {
		utils.Fatalf("Failed to write Poset genesis state: %v", err)
	}

	err = dbs.Flush(genesisAtropos.Bytes())
	if err != nil {
		utils.Fatalf("Failed to flush genesis state: %v", err)
	}

	if isNew {
		log.Info("Applied genesis state", "hash", cdb.GetGenesisHash().String())
	} else {
		log.Info("Genesis state is already written", "hash", cdb.GetGenesisHash().String())
	}

	// create consensus
	engine := poset.New(gossipCfg.Net.Dag, cdb, gdb)

	// Check DB integration
	checkDbIntegration(engine, adb, gdb)

	return engine, adb, gdb
}

// SetAccountKey sets key into accounts manager and unlocks it with pswd.
func SetAccountKey(
	am *accounts.Manager, key *ecdsa.PrivateKey, pswd string,
) (
	acc accounts.Account,
) {
	kss := am.Backends(keystore.KeyStoreType)
	if len(kss) < 1 {
		log.Warn("Keystore is not found")
		return
	}
	ks := kss[0].(*keystore.KeyStore)

	acc = accounts.Account{
		Address: crypto.PubkeyToAddress(key.PublicKey),
	}

	imported, err := ks.ImportECDSA(key, pswd)
	if err == nil {
		acc = imported
	} else if err.Error() != "account already exists" {
		log.Crit("Failed to import key", "err", err)
	}

	err = ks.Unlock(acc, pswd)
	if err != nil {
		log.Crit("failed to unlock key", "err", err)
	}

	return
}

func checkDbIntegration(engine *poset.Poset, adb *app.Store, gdb *gossip.Store) {
	lastEpoch := engine.GetEpoch()

	// get top events
	topEvents := gdb.GetHeads(lastEpoch)

	topEventsCount := 0
	topEventsMap := make(map[hash.Event]*inter.Event)
	events := make([]*inter.Event, 0, len(topEvents))
	eventsByNodes := make(map[idx.StakerID][]*inter.Event)

	// get all events in epoch
	gdb.ForEachEvent(lastEpoch, func(e *inter.Event)bool{
		// Save events without parents in list for compare with topEvents
		if e == nil {
			return false
		}
		events = append(events, e)
		if _, ok := eventsByNodes[e.Creator]; !ok {
			eventsByNodes[e.Creator] = make([]*inter.Event, 0, 1)
		}
		eventsByNodes[e.Creator] = append(eventsByNodes[e.Creator], e)

		// detect head (top) events
		topEventsMap[e.Hash()] = e
		if e.Parents != nil || len(e.Parents) > 0 {
			for _, pe := range e.Parents {
				delete(topEventsMap, pe)
			}
		}
		return true
	})
	topEventsCount = len(topEventsMap)

	// Compare topEvents set == topEventsFromList
	if len(topEvents) != topEventsCount {
		log.Crit("check db integration: root events count from GetHeads not equal root events from ForEachEvent")
	}
	for _, e := range topEvents {
		_, ok := topEventsMap[e]
		if !ok {
			log.Crit("check db integration: root event from GetHeads absent in events from ForEachEvent")
		}
	}

	// Check lamports
	if events[0].Lamport != 1 {
		log.Crit("check db integration: lamport at first event in epoch not equal 1")
	}
	lastLamport := idx.Lamport(0)
	for _, e := range events {
		if e.Lamport - lastLamport > 1 {
			log.Crit("check db integration: lamport between two events great then 1")
		}
		lastLamport = e.Lamport
	}

	// check seq by nodes
	for _, l := range eventsByNodes {
		if events[0].Seq != 1 {
			log.Crit("check db integration: seq at first event at one creator not equal 1")
		}
		lastSeq := idx.Event(0)
		for _, e := range l {
			if e.Seq - lastSeq > 1 {
				log.Crit("check db integration: seq between two events with one creator great then 1")
			}
		}
	}
}
