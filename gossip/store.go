package gossip

import (
	"bytes"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hashicorp/golang-lru"

	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
	"github.com/Fantom-foundation/go-lachesis/kvdb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/flushable"
	"github.com/Fantom-foundation/go-lachesis/kvdb/memorydb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/no_key_is_err"
	"github.com/Fantom-foundation/go-lachesis/kvdb/table"
	"github.com/Fantom-foundation/go-lachesis/logger"
)

// Store is a node persistent storage working over physical key-value database.
type Store struct {
	dbs *flushable.SyncedPool
	cfg StoreConfig

	mainDb kvdb.KeyValueStore
	table  struct {
		Peers     kvdb.KeyValueStore `table:"peer"`
		Events    kvdb.KeyValueStore `table:"event"`
		Blocks    kvdb.KeyValueStore `table:"block"`
		PackInfos kvdb.KeyValueStore `table:"packinfo"`
		Packs     kvdb.KeyValueStore `table:"pack"`
		PacksNum  kvdb.KeyValueStore `table:"packsnum"`

		ActiveValidatorScores kvdb.KeyValueStore `table:"actvscore"`
		DirtyValidatorScores  kvdb.KeyValueStore `table:"drtvscore"`
		BlockParticipation    kvdb.KeyValueStore `table:"blockprtcp"`
		Delegates			  kvdb.KeyValueStore `table:"delegates"`
		incMutex              *sync.Mutex

		// API-only tables
		BlockHashes        kvdb.KeyValueStore `table:"blockh"`
		Receipts           kvdb.KeyValueStore `table:"receipts"`
		TxPositions        kvdb.KeyValueStore `table:"txp"`
		ScoreCheckpoint    kvdb.KeyValueStore `table:"schekpoint"`
		ValidatorPOIScore  kvdb.KeyValueStore `table:"valpoiscore"`
		AddressGasUsed     kvdb.KeyValueStore `table:"addrgasused"`
		AddressLastTrxTime kvdb.KeyValueStore `table:"addrlasttrxtm"`
		TotalPOIGasUsed    kvdb.KeyValueStore `table:"poigasused"`

		TmpDbs kvdb.KeyValueStore `table:"tmpdbs"`

		Evm      ethdb.Database
		EvmState state.Database
	}

	cache struct {
		Events             *lru.Cache `cache:"-"` // store by pointer
		EventsHeaders      *lru.Cache `cache:"-"` // store by pointer
		Blocks             *lru.Cache `cache:"-"` // store by pointer
		PackInfos          *lru.Cache `cache:"-"` // store by value
		Receipts           *lru.Cache `cache:"-"` // store by value
		TxPositions        *lru.Cache `cache:"-"` // store by pointer
		BlockParticipation *lru.Cache `cache:"-"` // store by pointer
		BlockHashes        *lru.Cache `cache:"-"` // store by pointer
		ScoreCheckpoint    *lru.Cache `cache:"-"` // store by pointer
	}

	tmpDbs

	logger.Instance
}

// NewMemStore creates store over memory map.
func NewMemStore() *Store {
	mems := memorydb.NewProducer("")
	dbs := flushable.NewSyncedPool(mems)
	cfg := LiteStoreConfig()

	return NewStore(dbs, cfg)
}

// NewStore creates store over key-value db.
func NewStore(dbs *flushable.SyncedPool, cfg StoreConfig) *Store {
	s := &Store{
		dbs:      dbs,
		cfg:      cfg,
		mainDb:   dbs.GetDb("gossip-main"),
		Instance: logger.MakeInstance(),
	}

	table.MigrateTables(&s.table, s.mainDb)

	evmTable := no_key_is_err.Wrap(table.New(s.mainDb, []byte("evm_"))) // ETH expects that "not found" is an error
	s.table.Evm = rawdb.NewDatabase(evmTable)
	s.table.EvmState = state.NewDatabase(s.table.Evm)
	s.table.incMutex = &sync.Mutex{}

	s.initTmpDbs()
	s.initCache()

	return s
}

func (s *Store) initCache() {
	s.cache.Events = s.makeCache(s.cfg.EventsCacheSize)
	s.cache.EventsHeaders = s.makeCache(s.cfg.EventsHeadersCacheSize)
	s.cache.Blocks = s.makeCache(s.cfg.BlockCacheSize)
	s.cache.PackInfos = s.makeCache(s.cfg.PackInfosCacheSize)
	s.cache.Receipts = s.makeCache(s.cfg.ReceiptsCacheSize)
	s.cache.TxPositions = s.makeCache(s.cfg.TxPositionsCacheSize)
	s.cache.BlockParticipation = s.makeCache(64)
	s.cache.BlockHashes = s.makeCache(s.cfg.BlockCacheSize)
	s.cache.ScoreCheckpoint = s.makeCache(4)
}

// Close leaves underlying database.
func (s *Store) Close() {
	setnil := func() interface{} {
		return nil
	}

	table.MigrateTables(&s.table, nil)
	table.MigrateCaches(&s.cache, setnil)

	s.mainDb.Close()
}

// Commit changes.
func (s *Store) Commit(flushID []byte, immediately bool) error {
	if flushID == nil {
		// if flushId not specified, use current time
		buf := bytes.NewBuffer(nil)
		buf.Write([]byte{0xbe, 0xee})                                    // 0xbeee eyecatcher that flushed time
		buf.Write(bigendian.Int64ToBytes(uint64(time.Now().UnixNano()))) // current UNIX time
		flushID = buf.Bytes()
	}

	if immediately {
		return s.dbs.Flush(flushID)
	}

	_, err := s.dbs.FlushIfNeeded(flushID)
	return err
}

// StateDB returns state database.
func (s *Store) StateDB(from common.Hash) *state.StateDB {
	db, err := state.New(common.Hash(from), s.table.EvmState)
	if err != nil {
		s.Log.Crit("Failed to open state", "err", err)
	}
	return db
}

/*
 * Utils:
 */

// set RLP value
func (s *Store) set(table kvdb.KeyValueStore, key []byte, val interface{}) {
	buf, err := rlp.EncodeToBytes(val)
	if err != nil {
		s.Log.Crit("Failed to encode rlp", "err", err)
	}

	if err := table.Put(key, buf); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// get RLP value
func (s *Store) get(table kvdb.KeyValueStore, key []byte, to interface{}) interface{} {
	buf, err := table.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if buf == nil {
		return nil
	}

	err = rlp.DecodeBytes(buf, to)
	if err != nil {
		s.Log.Crit("Failed to decode rlp", "err", err)
	}
	return to
}

func (s *Store) has(table kvdb.KeyValueStore, key []byte) bool {
	res, err := table.Has(key)
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}
	return res
}

func (s *Store) makeCache(size int) *lru.Cache {
	if size <= 0 {
		return nil
	}

	cache, err := lru.New(size)
	if err != nil {
		s.Log.Crit("Error create LRU cache", "err", err)
		return nil
	}
	return cache
}
