package utils

// BitArray stores only bits of count number of int.
type BitArray struct {
	bits  uint
	count uint
	vals  []int
}

// NewBitArray makes bits array of int.
func NewBitArray(bits, count uint) *BitArray {
	if bits >= 8 {
		panic("too big size, use bytes")
	}

	return &BitArray{
		bits:  bits,
		count: count,
		vals:  make([]int, 0, count),
	}
}

// Size is a bytes count.
func (a *BitArray) Size() int {
	bits := a.bits * a.count

	s := bits / 8
	if bits%8 > 0 {
		s++
	}

	return int(s)
}

// Push bits of int into array.
func (a *BitArray) Push(v int) {
	if v < 0 {
		panic("positives only")
	}
	if v >= (1 << a.bits) {
		panic("too big value")
	}
	if uint(len(a.vals)) >= a.count {
		panic("count is exceeded")
	}

	a.vals = append(a.vals, v)
}

// Bytes from all bits.
func (a *BitArray) Bytes() []byte {
	var (
		raw []byte
		buf uint16
		n   uint
	)
	for _, v := range a.vals {
		buf += uint16(v << n)
		n += a.bits
		for n >= 8 {
			raw = append(raw, byte(buf))
			buf = buf >> 8
			n -= 8
		}
	}
	if n > 0 {
		raw = append(raw, byte(buf))
	}

	return raw
}

// Parse bits from bytes.
func (a *BitArray) Parse(raw []byte) []int {
	var (
		mask = uint16(1<<a.bits) - 1
		vals []int
		buf  uint16
		n    uint
	)
	for _, v := range raw {
		buf += uint16(v) << n
		n += 8
		for n >= a.bits && uint(len(vals)) < a.count {
			v := int(buf & mask)
			vals = append(vals, v)
			buf = (buf >> a.bits)
			n -= a.bits
		}
	}

	if uint(len(vals)) < a.count {
		panic("need more bytes")
	}

	return vals
}
