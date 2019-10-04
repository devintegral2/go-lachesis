package super_db

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestSuperDb(t *testing.T) {
	assertar := assert.New(t)
	datadir, err := ioutil.TempDir("", "dbspool")
	assertar.NoError(err)
	name0 := "dbname-0"
	name1 := "dbname-1"
	name2 := "dbname-2"

	pool, err := New(datadir)
	assertar.NoError(err)

	db0 := pool.GetDb(name0)
	db1 := pool.GetDb(name1)
	db2 := pool.GetDb(name2)
	assertar.NoError(db0.Put([]byte("key0"), []byte("value0")))
	assertar.NoError(db1.Put([]byte("key1"), []byte("value1")))
	assertar.NoError(db2.Put([]byte("key2"), []byte("value2")))

	// not exists until the flush
	for i, name := range []string{name0, name1, name2} {
		assertar.Nil(pool.bareDbs[name])
		assertar.NotNil(pool.wrappers[name])
		assertar.Equal(datadir + fmt.Sprintf("/dbname-%d-ldb", i), pool.pathes[name])
		_, err = os.Stat(pool.pathes[name])
		assertar.Error(err)
		assertar.True(os.IsNotExist(err))
	}

	// erase one of dbs before flush
	pool.DropDb(name1)

	// flush
	assertar.NoError(pool.Flush([]byte("id0")))

	// dbs are created now, but not name1
	_, err = os.Stat(pool.pathes[name0])
	assertar.NoError(err)
	_, err = os.Stat(pool.pathes[name1])
	assertar.Error(err)
	_, err = os.Stat(pool.pathes[name2])
	assertar.NoError(err)

	assertar.NoError(pool.CheckDbsSynced())
	assertar.Equal(db0, pool.GetDb(name0))
	assertar.NotEqual(db1, pool.GetDb(name1))
	assertar.Equal(db2, pool.GetDb(name2))

	// close
	assertar.NoError(pool.CloseAll())

	// re-open
	pool, err = New(datadir)
	assertar.NoError(pool.CheckDbsSynced())
	assertar.NoError(err)
	db2 = pool.GetLastDb("dbname")
	val, err := db2.Get([]byte("key2"))
	assertar.NoError(err)
	assertar.Equal([]byte("value2"), val)

	// drop db2
	pool.DropDb(name2)

	// flush
	assertar.NoError(pool.Flush([]byte("id1")))

	// db2 id dropped, db0 is fine
	_, err = os.Stat(pool.pathes[name0])
	assertar.NoError(err)
	_, err = os.Stat(pool.pathes[name1])
	assertar.Error(err)
	_, err = os.Stat(pool.pathes[name2])
	assertar.Error(err)
	assertar.NoError(pool.CheckDbsSynced())
	// close
	assertar.NoError(pool.CloseAll())
}
