package gossip

import (
	"time"

	"github.com/Fantom-foundation/go-lachesis/kvdb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/flushable"
	"github.com/Fantom-foundation/go-lachesis/kvdb/leveldb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/memorydb"
)

func cachedStore() *Store {
	mems := memorydb.NewProducer("", withDelay)
	dbs, _ := flushable.NewSyncedPool(mems)
	cfg := LiteStoreConfig()

	return NewStore(dbs, cfg)
}

func nonCachedStore() *Store {
	mems := memorydb.NewProducer("", withDelay)
	dbs, _ := flushable.NewSyncedPool(mems)
	cfg := StoreConfig{}

	return NewStore(dbs, cfg)
}

func realStore(dir string) *Store {
	disk := leveldb.NewProducer(dir)
	dbs, err := flushable.NewSyncedPool(disk)
	if err != nil {
		panic(err)
	}
	cfg := LiteStoreConfig()

	return NewStore(dbs, cfg)
}

func withDelay(db kvdb.KeyValueStore) kvdb.KeyValueStore {
	mem, ok := db.(*memorydb.Database)
	if ok {
		mem.SetDelay(time.Millisecond)

	}

	return db
}
