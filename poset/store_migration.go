package poset

import (
	"github.com/Fantom-foundation/go-lachesis/kvdb"
	"github.com/Fantom-foundation/go-lachesis/utils/migration"
)

func (s *Store) migrate() {
	versions := kvdb.NewIdProducer(s.table.Version, s.migrations())
	err := s.migrations().Exec(versions)
	if err != nil {
		s.Log.Crit("poset store migrations", "err", err)
	}
}

func (s *Store) migrations() *migration.Migration {
	return migration.Begin("lachesis-poset-store")
}