package super_db

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/go-lachesis/kvdb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/flushable"
	"github.com/Fantom-foundation/go-lachesis/kvdb/leveldb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/memorydb"
)

type SuperDb struct {
	pathes   map[string]string
	wrappers map[string]*flushable.Flushable
	bareDbs  map[string]kvdb.KeyValueStore

	queuedDrops map[string]struct{}

	prevFlushTime time.Time

	datadir string
	memonly bool

	mutex *sync.Mutex
}

func New(datadir string) (*SuperDb, error) {
	dirs, err := ioutil.ReadDir(datadir)
	if err != nil {
		return nil, err
	}

	sdb := &SuperDb{
		pathes:   make(map[string]string),
		wrappers: make(map[string]*flushable.Flushable),
		bareDbs:  make(map[string]kvdb.KeyValueStore),

		queuedDrops: make(map[string]struct{}),
		datadir:     datadir,
		mutex:       new(sync.Mutex),
	}

	for _, f := range dirs {
		dirname := f.Name()
		if f.IsDir() && strings.HasSuffix(dirname, "-ldb") {
			name := strings.TrimSuffix(dirname, "-ldb")
			path := filepath.Join(datadir, dirname)
			_, err := sdb.registerExisting(name, path)
			if err != nil {
				return nil, err
			}
		}
	}
	return sdb, nil
}

func NewMemOnly() *SuperDb {
	sdb := &SuperDb{
		pathes:   make(map[string]string),
		wrappers: make(map[string]*flushable.Flushable),
		bareDbs:  make(map[string]kvdb.KeyValueStore),

		queuedDrops: make(map[string]struct{}),
		memonly:     true,
		mutex:       new(sync.Mutex),
	}

	return sdb
}

func (sdb *SuperDb) registerExisting(name, path string) (kvdb.KeyValueStore, error) {
	db, err := openDb(path)
	if err != nil {
		return nil, err
	}
	wrapper := flushable.New(db)

	sdb.pathes[name] = path
	sdb.bareDbs[name] = db
	sdb.wrappers[name] = wrapper
	delete(sdb.queuedDrops, name)
	return wrapper, nil
}

func (sdb *SuperDb) registerNew(name, path string) kvdb.KeyValueStore {
	wrapper := flushable.New(memorydb.New())

	sdb.pathes[name] = path
	sdb.wrappers[name] = wrapper
	delete(sdb.bareDbs, name)
	delete(sdb.queuedDrops, name)
	return wrapper
}

func (sdb *SuperDb) GetDbByIndex(prefix string, index int64) kvdb.KeyValueStore {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	return sdb.getDb(fmt.Sprintf("%s-%d", prefix, index))
}

func (sdb *SuperDb) GetDb(name string) kvdb.KeyValueStore {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	return sdb.getDb(name)
}

func (sdb *SuperDb) getDb(name string) kvdb.KeyValueStore {
	if wrapper := sdb.wrappers[name]; wrapper != nil {
		return wrapper
	}
	return sdb.registerNew(name, filepath.Join(sdb.datadir, name+"-ldb"))
}

func (sdb *SuperDb) GetLastDb(prefix string) kvdb.KeyValueStore {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	options := make(map[string]int64)
	for name := range sdb.wrappers {
		if strings.HasPrefix(name, prefix) {
			s := strings.Split(name, "-")
			if len(s) < 2 {
				continue
			}
			indexStr := s[len(s)-1]
			index, err := strconv.ParseInt(indexStr, 10, 64)
			if err != nil {
				continue
			}
			options[name] = index
		}
	}
	if len(options) == 0 {
		return nil
	}

	maxIndexName := ""
	maxIndex := int64(math.MinInt64)
	for name, index := range options {
		if index > maxIndex {
			maxIndex = index
			maxIndexName = name
		}
	}

	return sdb.getDb(maxIndexName)
}

func (sdb *SuperDb) DropDb(name string) {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	if db := sdb.bareDbs[name]; db == nil {
		// this DB wasn't flushed, just erase it from RAM then, and that's it
		sdb.erase(name)
		return
	}
	sdb.queuedDrops[name] = struct{}{}
}

func (sdb *SuperDb) erase(name string) {
	delete(sdb.wrappers, name)
	delete(sdb.pathes, name)
	delete(sdb.bareDbs, name)
	delete(sdb.queuedDrops, name)
}

func (sdb *SuperDb) Flush(id []byte) error {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	return sdb.flush(id)
}

func (sdb *SuperDb) flush(id []byte) error {
	if sdb.memonly {
		return nil
	}

	key := []byte("flag")

	// drop old DBs
	for name := range sdb.queuedDrops {
		db := sdb.bareDbs[name]
		if db != nil {
			err := db.Close()
			if err != nil {
				return err
			}
			db.Drop()
		}
		sdb.erase(name)
	}

	// create new DBs, which were not dropped
	for name, wrapper := range sdb.wrappers {
		if db := sdb.bareDbs[name]; db == nil {
			db, err := openDb(sdb.pathes[name])
			if err != nil {
				return err
			}
			err = db.Put(key, []byte("initial")) // first clean flag
			if err != nil {
				return err
			}

			sdb.bareDbs[name] = db
			wrapper.SetUnderlyingDB(db)
		}
	}

	// write dirty flags
	for _, db := range sdb.bareDbs {
		marker := bytes.NewBuffer(nil)
		prev, err := db.Get(key)
		if err != nil {
			return err
		}
		if prev == nil {
			return errors.New("not found prev flushed state marker")
		}

		marker.Write([]byte("dirty"))
		marker.Write(prev)
		marker.Write(id)
		err = db.Put(key, marker.Bytes())
		if err != nil {
			return err
		}
	}

	// flush data
	for _, wrapper := range sdb.wrappers {
		err := wrapper.Flush()
		if err != nil {
			return err
		}
	}

	// write clean flags
	for _, wrapper := range sdb.wrappers {
		err := wrapper.Put(key, id)
		if err != nil {
			return err
		}
		err = wrapper.Flush()
		if err != nil {
			return err
		}
	}

	sdb.prevFlushTime = time.Now()
	return nil
}

func (sdb *SuperDb) FlushIfNeeded(id []byte) (bool, error) {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	if time.Since(sdb.prevFlushTime) > 10*time.Minute {
		err := sdb.Flush(id)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	totalNotFlushed := 0
	for _, db := range sdb.wrappers {
		totalNotFlushed += db.NotFlushedSizeEst()
	}

	if totalNotFlushed > 100*1024*1024 {
		err := sdb.Flush(id)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// call on startup, after all dbs are registered
func (sdb *SuperDb) CheckDbsSynced() error {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	key := []byte("flag")
	var prevId *[]byte
	for _, db := range sdb.bareDbs {
		mark, err := db.Get(key)
		if err != nil {
			return err
		}
		if bytes.HasPrefix(mark, []byte("dirty")) {
			return errors.New("dirty")
		}
		if prevId == nil {
			prevId = &mark
		}
		if bytes.Compare(mark, *prevId) != 0 {
			return errors.New("not synced")
		}
	}
	return nil
}

func (sdb *SuperDb) CloseAll() error {
	sdb.mutex.Lock()
	defer sdb.mutex.Unlock()

	for _, db := range sdb.bareDbs {
		if err := db.Close(); err != nil {
			return err
		}
	}
	return nil
}

func openDb(path string) (
	db kvdb.KeyValueStore,
	err error,
) {
	var stopWatcher func()

	onClose := func() error {
		if stopWatcher != nil {
			stopWatcher()
		}
		return nil
	}
	onDrop := func() error {
		return os.RemoveAll(path)
	}

	db, err = leveldb.New(path, 16, 0, "", onClose, onDrop)
	if err != nil {
		return nil, err
	}

	// TODO: dir watcher instead of file watcher needed.
	//stopWatcher = metrics.StartFileWatcher(name+"_db_file_size", f)

	return
}
