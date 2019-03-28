package posnode

import (
	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"

	"github.com/Fantom-foundation/go-lachesis/src/common"
	"github.com/Fantom-foundation/go-lachesis/src/kvdb"
	"github.com/Fantom-foundation/go-lachesis/src/posnode/wire"
)

// Store is a node persistent storage working over physical key-value database.
type Store struct {
	physicalDB kvdb.Database

	peers        kvdb.Database
	top10PeersID kvdb.Database
	knownHeights kvdb.Database
}

// NewMemStore creates store over memory map.
func NewMemStore() *Store {
	s := &Store{
		physicalDB: kvdb.NewMemDatabase(),
	}
	s.init()
	return s
}

// NewBadgerStore creates store over badger database.
func NewBadgerStore(db *badger.DB) *Store {
	s := &Store{
		physicalDB: kvdb.NewBadgerDatabase(db),
	}
	s.init()
	return s
}

func (s *Store) init() {
	s.peers = kvdb.NewTable(s.physicalDB, "peer_")
	s.top10PeersID = kvdb.NewTable(s.physicalDB, "top10PeersID_")
	s.knownHeights = kvdb.NewTable(s.physicalDB, "knownHeights_")
}

// Close leaves underlying database.
func (s *Store) Close() {
	s.peers = nil
	s.top10PeersID = nil
	s.knownHeights = nil
	s.physicalDB.Close()
}

// SetTopPeersID stores peers ID.
func (s *Store) SetTopPeersID(ids []common.Address) {
	length := len(ids)

	if length > 10 {
		panic("Error: size of array more than 10")
	}

	addresses := []string{}

	// TODO: too slow solution
	for _, id := range ids {
		addresses = append(addresses, id.Hex())
	}

	w := &wire.PeersID{
		IDs: addresses,
	}

	s.set(s.top10PeersID, []byte{0}, w)
}

// GetTopPeersID returns stored peer.
func (s *Store) GetTopPeersID() *[]common.Address {
	peersID, _ := s.get(s.top10PeersID, []byte{0}, &wire.PeersID{}).(*wire.PeersID)

	addresses := []common.Address{}

	// TODO: too slow solution
	for _, id := range peersID.IDs {
		addresses = append(addresses, common.HexToAddress(id))
	}

	return &addresses
}

// SetHeights stores known heights.
func (s *Store) SetHeights(heights *wire.KnownEvents) {
	s.set(s.knownHeights, []byte{0}, heights)
}

// GetHeights returns stored known heights.
func (s *Store) GetHeights() *wire.KnownEvents {
	heights, _ := s.get(s.knownHeights, []byte{0}, &wire.KnownEvents{}).(*wire.KnownEvents)
	return heights
}

// SetPeer stores peer.
func (s *Store) SetPeer(peer *Peer) {
	w := peer.ToWire()
	s.set(s.peers, peer.ID.Bytes(), w)
}

// GetPeerInfo returns stored peer info.
// Result is a ready gRPC message.
func (s *Store) GetPeerInfo(id common.Address) *wire.PeerInfo {
	w, _ := s.get(s.peers, id.Bytes(), &wire.PeerInfo{}).(*wire.PeerInfo)
	return w
}

// GetPeer returns stored peer.
func (s *Store) GetPeer(id common.Address) *Peer {
	w := s.GetPeerInfo(id)
	if w == nil {
		return nil
	}

	return WireToPeer(w)
}

/*
 * Utils:
 */

func (s *Store) set(table kvdb.Database, key []byte, val proto.Message) {
	var pbf proto.Buffer

	if err := pbf.Marshal(val); err != nil {
		panic(err)
	}

	if err := table.Put(key, pbf.Bytes()); err != nil {
		panic(err)
	}
}

func (s *Store) get(table kvdb.Database, key []byte, to proto.Message) proto.Message {
	buf, err := table.Get(key)
	if err != nil {
		panic(err)
	}
	if buf == nil {
		return nil
	}

	err = proto.Unmarshal(buf, to)
	if err != nil {
		panic(err)
	}
	return to
}

func (s *Store) has(table kvdb.Database, key []byte) bool {
	res, err := table.Has(key)
	if err != nil {
		panic(err)
	}
	return res
}