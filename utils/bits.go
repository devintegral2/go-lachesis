package utils

// BitArray stores only bits of count number of int.
type BitArray struct {
	bits   uint
	count  uint
	vals   []int
	offset uint
	size   int
}

// NewBitArray makes bits array of int.
func NewBitArray(bits, count uint) *BitArray {
	if bits >= 8 {
		panic("too big size, use bytes")
	}

	return &BitArray{
		bits:  bits,
		count: count,
		vals:  make([]int, count, count),
		size:  calcSize(bits, count),
	}
}

func calcSize(bits, count uint) int {
	bits = bits * count
	s := bits / 8
	if bits%8 > 0 {
		s++
	}
	return int(s)
}

// Size is a bytes count.
func (a *BitArray) Size() int {
	return a.size
}

// Push bits of int into array.
func (a *BitArray) Push(v int) {
	if v < 0 {
		panic("positives only")
	}
	if v >= (1 << a.bits) {
		panic("too big value")
	}

	a.vals[a.offset] = v
	a.offset++
}

// Pop int from array.
func (a *BitArray) Pop() int {
	v := a.vals[a.offset]
	a.offset++
	return v
}

// Bytes from all bits.
func (a *BitArray) Bytes() []byte {
	if a.offset < a.count {
		panic("array is not full yet")
	}

	var (
		raw = make([]byte, a.size, a.size)
		i   int
		buf uint16
		n   uint
	)
	for _, v := range a.vals {
		buf += uint16(v << n)
		n += a.bits
		for n >= 8 {
			raw[i] = byte(buf)
			i++
			buf = buf >> 8
			n -= 8
		}
	}
	if n > 0 {
		raw[i] = byte(buf)
	}

	return raw
}

// Parse bits from bytes.
func (a *BitArray) Parse(raw []byte) {
	if len(raw) != a.Size() {
		panic("need <.Size()> bytes")
	}

	var (
		mask = uint16(1<<a.bits) - 1
		i    uint
		buf  uint16
		n    uint
	)
	for _, v := range raw {
		buf += uint16(v) << n
		n += 8
		for n >= a.bits && i < a.count {
			a.vals[i] = int(buf & mask)
			i++
			buf = (buf >> a.bits)
			n -= a.bits
		}
	}

	a.offset = 0
}
