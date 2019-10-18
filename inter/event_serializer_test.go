package inter

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

func TestEventHeaderData_EncodeRLP(t *testing.T) {
	assertar := assert.New(t)

	header0 := FakeEvent().EventHeaderData
	buf, err := rlp.EncodeToBytes(&header0)
	if !assertar.NoError(err) {
		return
	}

	var header1 EventHeaderData
	err = rlp.DecodeBytes(buf, &header1)
	if !assertar.NoError(err) {
		return
	}

	assert.EqualValues(t, header0, header1)
}

func BenchmarkEventHeaderData_EncodeRLP(b *testing.B) {
	header := FakeEvent().EventHeaderData

	// TODO: for go1.13+
	// b.ReportMetric(float64(len(buf)), "Bytes")

	for i := 0; i < b.N; i++ {
		_, err := rlp.EncodeToBytes(&header)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventHeaderData_DecodeRLP(b *testing.B) {
	header := FakeEvent().EventHeaderData

	buf, err := rlp.EncodeToBytes(&header)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &header)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func FakeEvent() *Event {
	var epoch idx.Epoch = 52123

	e := NewEvent()
	e.Epoch = epoch
	e.Seq = idx.Event(9)
	e.Creator = hash.FakePeer()
	e.Parents = FakeEventHashes(epoch, 8)
	e.Extra = make([]byte, 10, 10)
	e.Sig = []byte{}

	_, err := rand.Read(e.Extra)
	if err != nil {
		panic(err)
	}

	return e
}

// FakeEventHashes generates random event hashes for testing purpose.
func FakeEventHashes(epoch idx.Epoch, n int) hash.Events {
	res := hash.Events{}
	for i := 0; i < n; i++ {
		res.Add(FakeEventHash(epoch))
	}
	return res
}

// FakeEventHash generates random event hash for testing purpose.
func FakeEventHash(epoch idx.Epoch) (h hash.Event) {
	_, err := rand.Read(h[:])
	if err != nil {
		panic(err)
	}
	copy(h[0:4], bigendian.Int32ToBytes(uint32(epoch)))
	return
}
