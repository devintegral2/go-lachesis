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
	bits := uint(4) // int64/8 = 8 (bytes count), could be stored in 4 bits
	header := utils.NewBitArray(bits, fcount)

	headerBytes := 1 + // header length
		header.Size()

	maxBytes := headerBytes +
		len(fields32)*4 +
		len(fields64)*8 +
		len(e.Parents)*(32-4) + // without idx.Epoch
		common.AddressLength + // Creator
		common.HashLength + // PrevEpochHash
		common.HashLength + // TxHash
		len(e.Extra)
	raw := make([]byte, maxBytes, maxBytes)
	rawHeader := raw[:headerBytes]
	rawBody := raw[headerBytes:]

	rawHeader[0] = byte(header.Size())
	buf := fast.NewBuffer(&rawBody)
	for _, f := range fields32 {
		n := writeUint32Compact(buf, f)
		header.Push(n)
	}
	for _, f := range fields64 {
		n := writeUint64Compact(buf, f)
		header.Push(n)
	}
	for _, f := range fieldsBool {
		if f {
			header.Push(1)
		} else {
			header.Push(0)
		}
	}
	copy(rawHeader[1:], header.Bytes())

	for _, p := range e.Parents {
		buf.Write(p.Bytes()[4:]) // without epoch
	}

	buf.Write(e.Creator.Bytes())
	buf.Write(e.PrevEpochHash.Bytes())
	buf.Write(e.TxHash.Bytes())
	buf.Write(e.Extra)

	length := headerBytes + buf.Position()
	return raw[:length], nil
}

func writeUint32Compact(buf *fast.Buffer, v uint32) (bytes int) {
	for v > 0 {
		buf.WriteByte(byte(v))
		bytes++
		v = v >> 8
	}
	return
}

func writeUint64Compact(buf *fast.Buffer, v uint64) (bytes int) {
	for v > 0 {
		buf.WriteByte(byte(v))
		bytes++
		v = v >> 8
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

	buf := fast.NewBuffer(&raw)

	fcount := uint(len(fields32) + len(fields64) + len(fieldsBool))
	bits := uint(4) // int64/8 = 8 (bytes count), could be stored in 4 bits
	header := utils.NewBitArray(bits, fcount)
	headerBytes := int(buf.Read(1)[0])
	n := 0
	nn := header.Parse(buf.Read(headerBytes))

	for _, f := range fields32 {
		n, nn = nn[0], nn[1:]
		*f = readUint32Compact(buf, n)
	}
	for _, f := range fields64 {
		n, nn = nn[0], nn[1:]
		*f = readUint64Compact(buf, n)
	}
	for _, f := range fieldsBool {
		n, nn = nn[0], nn[1:]
		*f = (n != 0)
	}

	e.Parents = hash.Events{}
	for i := uint32(0); i < parentCount; i++ {
		tail := buf.Read(common.HashLength - 4) // without epoch
		bb := append(e.Epoch.Bytes(), tail...)
		p := hash.BytesToEvent(bb)
		e.Parents.Add(p)
	}

	e.Creator = common.BytesToAddress(buf.Read(common.AddressLength))
	e.PrevEpochHash = common.BytesToHash(buf.Read(common.HashLength))
	e.TxHash = common.BytesToHash(buf.Read(common.HashLength))
	e.Extra = buf.Read(len(raw) - buf.Position())

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
