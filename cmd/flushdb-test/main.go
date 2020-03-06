package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
	"github.com/Fantom-foundation/go-lachesis/kvdb/flushable"
	"github.com/Fantom-foundation/go-lachesis/kvdb/leveldb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/table"
)

func init() {
	rand.Seed(time.Now().Unix())

	log.Root().SetHandler(
		log.LvlFilterHandler(log.LvlWarn,
			log.StdoutHandler))
}

func main() {
	dir := "/tmp/flushdb-test"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	defer func() {
		if r := recover(); r != nil {
			log.Warn("<< panic exit", "err", r)
			_ = os.RemoveAll(dir)
		}
	}()

	err := os.MkdirAll(dir, 0760)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	dbs := leveldb.NewProducer(dir)
	pool, err := flushable.NewSyncedPool(dbs)
	if err != nil {
		log.Warn(">> detected inconsistency", "err", err)
		_ = os.RemoveAll(dir)
		return
	}

	last, err := checkConsistency(pool)
	if err != nil {
		log.Warn(">> undetected inconsistency", "err", err)
		os.Exit(1)
	}

	log.Warn(">> all right")

	gracefulExit := make(chan struct{})
	notify := make(chan struct{}, 1)

	go func() {
		for range notify {
			x := rand.Intn(50)
			switch {
			case x < 3:
				log.Warn("<< instant exit")
				os.Exit(0)
			case x < 5:
				log.Warn("<< graceful exit")
				close(gracefulExit)
				return
			default:
				continue
			}
		}
	}()

	for {
		select {
		case <-gracefulExit:
			return
		default:
			last = writeData(pool, last, 1000, notify)
		}
	}
}

const (
	dbName    = "db1"
	tableName = "t1"
)

func checkConsistency(pool *flushable.SyncedPool) (last uint64, err error) {
	db := pool.GetDb(dbName)
	t := table.New(db, []byte(tableName))

	it := t.NewIterator()
	defer it.Release()

	var prev uint64
	last = prev

	for it.Next() {
		last = bigendian.BytesToInt64(it.Key())
		if last != prev+1 {
			err = fmt.Errorf("key inconsistency: %d --> %d",
				prev, last)
			return
		}

		exp := key2val(it.Key())
		if !bytes.Equal(exp, it.Value()) {
			err = fmt.Errorf("val inconsistency: %d --> %d",
				prev, last)
			return
		}

		prev = last
	}

	err = it.Error()
	if err != nil {
		panic(err)
	}

	return
}

func writeData(pool *flushable.SyncedPool, start uint64, count int, notify chan<- struct{}) (last uint64) {
	db := pool.GetDb(dbName)
	t := table.New(db, []byte(tableName))

	var err error
	last = start
	for ; count > 0; count-- {
		last++
		key := bigendian.Int64ToBytes(last)
		val := key2val(key)

		err = t.Put(key, val)
		if err != nil {
			panic(err)
		}
	}

	notify <- struct{}{}
	err = pool.Flush(bigendian.Int64ToBytes(start))
	if err != nil {
		panic(err)
	}

	return
}

func key2val(key []byte) (val []byte) {
	const n = 100

	h := md5.New()
	_, err := h.Write(key)
	if err != nil {
		panic(err)
	}
	hash := h.Sum(nil)

	val = make([]byte, 0, n*len(hash))
	for i := 0; i < n; i++ {
		val = append(val, hash...)
	}

	return
}
