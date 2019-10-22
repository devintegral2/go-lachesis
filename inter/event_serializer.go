package inter

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/utils"
	"github.com/Fantom-foundation/go-lachesis/utils/fast"
)

// EncodeRLP implements rlp.Encoder interface.
func (e *EventHeaderData) EncodeRLP(w io.Writer) error {
	bytes, err := e.MarshalBinary()
	if err != nil {
		return err
	}

	err = rlp.Encode(w, &bytes)

	return err
}

// DecodeRLP implements rlp.Decoder interface.
func (e *EventHeaderData) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	err = e.UnmarshalBinary(bytes)

	return err
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (e *EventHeaderData) MarshalBinary() ([]byte, error) {
	fields32 := []uint32{
		e.Version,
		uint32(e.Epoch),
		uint32(e.Seq),
		uint32(e.Frame),
		uint32(e.Lamport),
		uint32(len(e.Parents)),
	}
	fields64 := []uint64{
		e.GasPowerLeft,
		e.GasPowerUsed,
		uint64(e.ClaimedTime),
		uint64(e.MedianTime),
	}
	fieldsBool := []bool{
		e.IsRoot,
	}

	fcount := uint(len(fields32) + len(fields64) + len(fieldsBool))
	bits := uint(3) // int64/8 = 8 (bytes count) - 1 (zero bytes is not used), could be stored in 3 bits
	header := utils.NewBitArray(bits, fcount)

	maxBytes := header.Size() +
		len(fields32)*4 +
		len(fields64)*8 +
		len(e.Parents)*(32-4) + // without idx.Epoch
		common.AddressLength + // Creator
		common.HashLength + // PrevEpochHash
		common.HashLength + // TxHash
		len(e.Extra)

	raw := make([]byte, maxBytes, maxBytes)

	headerW := header.Writer(raw[0:header.Size()])
	buf := fast.NewBuffer(raw[header.Size():])

	for _, f := range fields32 {
		n := writeUint32Compact(buf, f)
		headerW.Push(n)
	}
	for _, f := range fields64 {
		n := writeUint64Compact(buf, f)
		headerW.Push(n)
	}
	for _, f := range fieldsBool {
		if f {
			headerW.Push(1)
		} else {
			headerW.Push(0)
		}
	}

	for _, p := range e.Parents {
		buf.Write(p.Bytes()[4:]) // without epoch
	}

	buf.Write(e.Creator.Bytes())
	buf.Write(e.PrevEpochHash.Bytes())
	buf.Write(e.TxHash.Bytes())
	buf.Write(e.Extra)

	length := header.Size() + buf.Position()
	return raw[:length], nil
}

func writeUint32Compact(buf *fast.Buffer, v uint32) (bytes int) {
	for {
		buf.WriteByte(byte(v))
		bytes++
		v = v >> 8
		if v == 0 {
			break
		}
	}
	return
}

func writeUint64Compact(buf *fast.Buffer, v uint64) (bytes int) {
	for {
		buf.WriteByte(byte(v))
		bytes++
		v = v >> 8
		if v == 0 {
			break
		}
	}
	return
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (e *EventHeaderData) UnmarshalBinary(raw []byte) error {
	var parentCount uint32

	fields32 := []*uint32{
		&e.Version,
		(*uint32)(&e.Epoch),
		(*uint32)(&e.Seq),
		(*uint32)(&e.Frame),
		(*uint32)(&e.Lamport),
		&parentCount,
	}
	fields64 := []*uint64{
		&e.GasPowerLeft,
		&e.GasPowerUsed,
		(*uint64)(&e.ClaimedTime),
		(*uint64)(&e.MedianTime),
	}
	fieldsBool := []*bool{
		&e.IsRoot,
	}

	fcount := uint(len(fields32) + len(fields64) + len(fieldsBool))
	bits := uint(3) // int64/8 = 8 (bytes count) - 1 (zero bytes is not used), could be stored in 3 bits
	header := utils.NewBitArray(bits, fcount)

	headerR := header.Reader(raw[:header.Size()])
	buf := fast.NewBuffer(raw[header.Size():])

	for _, f := range fields32 {
		n := headerR.Pop()
		*f = readUint32Compact(buf, n)
	}
	for _, f := range fields64 {
		n := headerR.Pop()
		*f = readUint64Compact(buf, n)
	}
	for _, f := range fieldsBool {
		n := headerR.Pop()
		*f = (n != 0)
	}

	e.Parents = make(hash.Events, parentCount, parentCount)
	for i := uint32(0); i < parentCount; i++ {
		copy(e.Parents[i][:4], e.Epoch.Bytes())
		copy(e.Parents[i][4:], buf.Read(common.HashLength-4)) // without epoch
	}

	e.Creator.SetBytes(buf.Read(common.AddressLength))
	e.PrevEpochHash.SetBytes(buf.Read(common.HashLength))
	e.TxHash.SetBytes(buf.Read(common.HashLength))
	e.Extra = buf.Read(len(raw) - header.Size() - buf.Position())

	return nil
}

func readUint32Compact(buf *fast.Buffer, bytes int) uint32 {
	var v uint32
	for i, b := range buf.Read(bytes) {
		v += uint32(b) << uint(8*i)
	}

	return v
}

func readUint64Compact(buf *fast.Buffer, bytes int) uint64 {
	var v uint64
	for i, b := range buf.Read(bytes) {
		v += uint64(b) << uint(8*i)
	}

	return v
}
