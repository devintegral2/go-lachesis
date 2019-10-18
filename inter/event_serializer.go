package inter

import (
	"io"
	//	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	//	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
	//	"github.com/Fantom-foundation/go-lachesis/common/littleendian"
	"github.com/Fantom-foundation/go-lachesis/hash"
	//	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/utils"
	"github.com/Fantom-foundation/go-lachesis/utils/fast"
)

const (
	EventHeaderFixedDataSize = 53
	SerializedCounterSize    = 4
)

func (e *EventHeaderData) EncodeRLP(w io.Writer) error {
	bytes, err := e.MarshalBinary()
	if err != nil {
		return err
	}

	err = rlp.Encode(w, &bytes)

	return err
}

func (e *EventHeaderData) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	err = e.UnmarshalBinary(bytes)

	return err
}

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

	raw[0] = byte(header.Size())
	buf := fast.NewBuffer(raw[headerBytes:])
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
	copy(raw[1:], header.Bytes())

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

	buf := fast.NewBuffer(raw)

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

	for i := uint32(0); i < parentCount; i++ {
		tail := buf.Read(common.HashLength - 4) // without epoch
		bb := append(e.Epoch.Bytes(), tail...)
		p := hash.BytesToEvent(bb)
		e.Parents.Add(p)
	}

	e.Creator = common.BytesToAddress(buf.Read(common.AddressLength))
	e.PrevEpochHash = common.BytesToHash(buf.Read(common.HashLength))
	e.TxHash = common.BytesToHash(buf.Read(common.HashLength))
	e.Extra = buf.Read(-1)

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

/*
func (e *EventHeaderData) encodeUint32FieldsToPacked(buf *fast_buffer.Buffer) {
	// Detect max value from 4 fields
	v1size := maxBytesForUint32(e.Version)
	v2size := maxBytesForUint32(uint32(e.Epoch))
	v3size := maxBytesForUint32(uint32(e.Seq))
	v4size := maxBytesForUint32(uint32(e.Frame))
	sizeByte := byte((v1size - 1) | ((v2size - 1) << 2) | ((v3size - 1) << 4) | ((v4size - 1) << 6))

	buf.Write([]byte{sizeByte})
	buf.Write(littleendian.Int32ToBytes(e.Version)[0:v1size])
	buf.Write(littleendian.Int32ToBytes(uint32(e.Epoch))[0:v2size])
	buf.Write(littleendian.Int32ToBytes(uint32(e.Seq))[0:v3size])
	buf.Write(littleendian.Int32ToBytes(uint32(e.Frame))[0:v4size])

	v1size = maxBytesForUint32(uint32(e.Lamport))
	sizeByte = byte(v1size - 1)
	buf.Write([]byte{sizeByte})
	buf.Write(littleendian.Int32ToBytes(uint32(e.Lamport))[0:v1size])
}

func (e *EventHeaderData) encodeUint64FieldsToPacked(buf *fast_buffer.Buffer) {
	v1size := maxBytesForUint64(e.GasPowerLeft)
	v2size := maxBytesForUint64(e.GasPowerUsed)
	sizeByte := byte((v1size - 1) | ((v2size - 1) << 4))

	buf.Write([]byte{sizeByte})
	buf.Write(littleendian.Int64ToBytes(e.GasPowerLeft)[0:v1size])
	buf.Write(littleendian.Int64ToBytes(e.GasPowerUsed)[0:v2size])

	v1size = maxBytesForUint64(uint64(e.ClaimedTime))
	v2size = maxBytesForUint64(uint64(e.MedianTime))
	sizeByte = byte((v1size - 1) | ((v2size - 1) << 4))
	buf.Write([]byte{sizeByte})
	buf.Write(littleendian.Int64ToBytes(uint64(e.ClaimedTime))[0:v1size])
	buf.Write(littleendian.Int64ToBytes(uint64(e.MedianTime))[0:v2size])
}

func (e *EventHeaderData) encodeParentsWithoutEpoch(buf *fast_buffer.Buffer) {
	// Sliced values
	parentsCount := len(e.Parents)
	buf.Write(littleendian.Int32ToBytes(uint32(parentsCount))[0:SerializedCounterSize])

	for _, ev := range e.Parents {
		buf.Write(ev.Bytes()[4:common.HashLength])
	}
}

func (e *EventHeaderData) decodePackedToUint32Fields(buf *fast_buffer.Buffer) {
	v1size, v2size, v3size, v4size := splitByteOn4Values(buf)

	e.Version = readPackedUint32(buf, v1size)
	e.Epoch = idx.Epoch(readPackedUint32(buf, v2size))
	e.Seq = idx.Event(readPackedUint32(buf, v3size))
	e.Frame = idx.Frame(readPackedUint32(buf, v4size))

	v1size, _, _, _ = splitByteOn4Values(buf)

	e.Lamport = idx.Lamport(readPackedUint32(buf, v1size))
}

func (e *EventHeaderData) decodePackedToUint64Fields(buf *fast_buffer.Buffer) {
	v1size, v2size := splitByteOn2Values(buf)

	e.GasPowerLeft = readPackedUint64(buf, v1size)
	e.GasPowerUsed = readPackedUint64(buf, v2size)

	v1size, v2size = splitByteOn2Values(buf)

	e.ClaimedTime = Timestamp(readPackedUint64(buf, v1size))
	e.MedianTime = Timestamp(readPackedUint64(buf, v2size))
}

func (e *EventHeaderData) decodeParentsWithoutEpoch(buf *fast_buffer.Buffer) {
	parentsCount := readUint32(buf)

	evBuf := make([]byte, common.HashLength)
	if parentsCount > 0 {
		// Get epoch from Epoch field
		copy(evBuf[0:4], bigendian.Int32ToBytes(uint32(e.Epoch)))
	}

	e.Parents = make(hash.Events, parentsCount, parentsCount)
	for i := 0; i < int(parentsCount); i++ {
		ev := hash.Event{}

		copy(evBuf[4:common.HashLength], buf.Read(common.HashLength-4))
		ev.SetBytes(evBuf)

		e.Parents[i] = ev
	}
}

func maxBytesForUint32(t uint32) uint {
	return maxBytesForUint64(uint64(t))
}

func maxBytesForUint64(t uint64) uint {
	mask := uint64(math.MaxUint64)
	for i := uint(1); i < 8; i++ {
		mask = mask << 8
		if mask&t == 0 {
			return i
		}
	}
	return 8
}

func splitByteOn4Values(buf *fast_buffer.Buffer) (v1size int, v2size int, v3size int, v4size int) {
	sizeByte := buf.Read(1)[0]
	v1size = int(sizeByte&3 + 1)
	v2size = int((sizeByte>>2)&3 + 1)
	v3size = int((sizeByte>>4)&3 + 1)
	v4size = int((sizeByte>>6)&3 + 1)

	return
}

func splitByteOn2Values(buf *fast_buffer.Buffer) (v1size int, v2size int) {
	sizeByte := buf.Read(1)[0]
	v1size = int(sizeByte&7 + 1)
	v2size = int((sizeByte>>4)&7 + 1)

	return
}

func readPackedUint32(buf *fast_buffer.Buffer, size int) uint32 {
	intBuf := []byte{0, 0, 0, 0}
	copy(intBuf, buf.Read(size))
	return littleendian.BytesToInt32(intBuf)
}

func readPackedUint64(buf *fast_buffer.Buffer, size int) uint64 {
	intBuf := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	copy(intBuf, buf.Read(size))
	return littleendian.BytesToInt64(intBuf)
}

func readByteBool(buf *fast_buffer.Buffer) bool {
	return buf.Read(1)[0] != byte(0)
}

func readUint32(buf *fast_buffer.Buffer) (data uint32) {
	return littleendian.BytesToInt32(buf.Read(4))
}
*/
