package poset

import (
	"bytes"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

// AddRoot stores the new root
func (s *Store) AddRoot(root *inter.Event) {
	key := bytes.Buffer{}
	key.Write(root.Frame.Bytes())
	key.Write(root.Creator.Bytes())
	key.Write(root.Hash().Bytes())

	if err := s.epochTable.Roots.Put(key.Bytes(), []byte{}); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// IsRoot returns true if event is root
func (s *Store) IsRoot(f idx.Frame, from common.Address, id hash.Event) bool {
	key := bytes.Buffer{}
	key.Write(f.Bytes())
	key.Write(from.Bytes())
	key.Write(id.Bytes())

	ok, err := s.epochTable.Roots.Has(key.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}
	return ok
}

const (
	frameSize   = 4
	addrSize    = 20
	eventIDSize = 32
)

type stopFlagType struct {
	stop bool
	lock *sync.Mutex
}

func NewStopFlag() *stopFlagType {
	return &stopFlagType{
		stop: false,
		lock: &sync.Mutex{},
	}
}

func (f *stopFlagType) IsStoped() bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.stop
}

func (f *stopFlagType) Stop() {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.stop = true
}

// ForEachRoot iterates all the roots in the specified frame
func (s *Store) ForEachRoot(parallel bool, f idx.Frame, do func(f idx.Frame, from common.Address, root hash.Event) bool) {

	it := s.epochTable.Roots.NewIteratorWithStart(f.Bytes())

	stop := NewStopFlag()
	wg := sync.WaitGroup{}

	defer it.Release()
	for !stop.IsStoped() && it.Next() {
		k := it.Key()
		wg.Add(1)

		block := func(key []byte) {
			defer wg.Done()

			if len(key) != frameSize+addrSize+eventIDSize {
				s.Log.Crit("Roots table: incorrect key len", "len", len(key))
			}
			actualF := idx.BytesToFrame(key[:frameSize])
			actualCreator := common.BytesToAddress(key[frameSize : frameSize+addrSize])
			actualID := hash.BytesToEvent(key[frameSize+addrSize:])
			if actualF < f {
				s.Log.Crit("Roots table: invalid frame", "frame", f, "expected", actualF)
			}

			if !stop.IsStoped() {
				if !do(actualF, actualCreator, actualID) {
					stop.Stop()
				}
			}
		}

		if parallel {
			go block(k)
		} else {
			block(k)
		}
	}
	wg.Wait()
	if it.Error() != nil {
		s.Log.Crit("Failed to iterate keys", "err", it.Error())
	}
}

// ForEachRootFrom iterates all the roots in the specified frame, from the specified validator
func (s *Store) ForEachRootFrom(f idx.Frame, from common.Address, do func(f idx.Frame, from common.Address, id hash.Event) bool) {
	prefix := append(f.Bytes(), from.Bytes()...)

	it := s.epochTable.Roots.NewIteratorWithPrefix(prefix)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		if len(key) != frameSize+addrSize+eventIDSize {
			s.Log.Crit("Roots table: incorrect key len", "len", len(key))
		}
		actualF := idx.BytesToFrame(key[:frameSize])
		actualCreator := common.BytesToAddress(key[frameSize : frameSize+addrSize])
		actualID := hash.BytesToEvent(key[frameSize+addrSize:])
		if actualF < f {
			s.Log.Crit("Roots table: invalid frame", "frame", f, "expected", actualF)
		}
		if actualCreator != from {
			s.Log.Crit("Roots table: invalid creator", "creator", from.String(), "expected", actualCreator.String())
		}

		if !do(actualF, actualCreator, actualID) {
			break
		}
	}
	if it.Error() != nil {
		s.Log.Crit("Failed to iterate keys", "err", it.Error())
	}
}
