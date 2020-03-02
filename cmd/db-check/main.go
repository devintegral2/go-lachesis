package main

import (
	"os"
	"path/filepath"

	"github.com/Fantom-foundation/go-lachesis/kvdb/leveldb"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	dir := filepath.Join(home, ".lachesis")

	if len(os.Args) >= 2 {
		dir = os.Args[1]
	}

	dbs := leveldb.NewProducer(dir)

	//checkPacks(dbs)
	//checkEvents(dbs)
	//checkAfterMigration(p)
	checkGasRefunds(dbs)

}
