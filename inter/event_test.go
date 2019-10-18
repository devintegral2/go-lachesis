package inter

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

func TestEventSerialization(t *testing.T) {
	assertar := assert.New(t)

	events := FakeEvents()
	for i, e0 := range events {
		dsc := fmt.Sprintf("iter#%d", i)

		buf, err := rlp.EncodeToBytes(e0)
		if !assertar.NoError(err, dsc) {
			break
		}

		assertar.Equal(len(buf), e0.Size())
		assertar.Equal(len(buf), e0.CalcSize())

		e1 := &Event{}
		err = rlp.DecodeBytes(buf, e1)
		if !assertar.NoError(err, dsc) {
			break
		}
		if e1.Sig == nil {
			e1.Sig = []uint8{}
		}

		assertar.Equal(len(buf), e1.CalcSize())
		assertar.Equal(len(buf), e1.Size())

		if !assertar.Equal(e0.EventHeader, e1.EventHeader, dsc) {
			break
		}
	}
}

func TestEventHash(t *testing.T) {
	var (
		events = FakeFuzzingEvents()
		hashes = make([]hash.Event, len(events))
	)

	t.Run("Calculation", func(t *testing.T) {
		for i, e := range events {
			hashes[i] = e.Hash()
		}
	})

	t.Run("Comparison", func(t *testing.T) {
		for i, e := range events {
			h := e.Hash()
			if h != hashes[i] {
				t.Fatal("Non-deterministic event hash detected")
			}
			for _, other := range hashes[i+1:] {
				if h == other {
					t.Fatal("Event hash collision detected")
				}
			}
		}
	})
}

func FakeEvents() (res []*Event) {
	var epoch idx.Epoch = 34245
	creators := []common.Address{
		{},
		hash.FakePeer(),
		hash.FakePeer(),
		hash.FakePeer(),
	}
	parents := []hash.Events{
		{},
		FakeEventHashes(epoch, 1),
		FakeEventHashes(epoch, 2),
		FakeEventHashes(epoch, 8),
	}
	i := 0
	for c := 0; c < len(creators); c++ {
		for p := 0; p < len(parents); p++ {
			e := NewEvent()
			e.Epoch = epoch
			e.Seq = idx.Event(p)
			e.Creator = creators[c]
			e.Parents = parents[p]
			e.Extra = []byte{}
			e.Sig = []byte{}

			res = append(res, e)
			i++
		}
	}
	return
}
