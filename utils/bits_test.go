package utils

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitArray(t *testing.T) {
	for i := uint(1); i < 8; i++ {
		testBitArray(t, i)
	}
}

func testBitArray(t *testing.T, bits uint) {
	expect := rand.Perm(1 << bits)
	count := len(expect)

	arr := NewBitArray(bits, uint(count))
	for _, v := range expect {
		arr.Push(v)
	}

	raw := arr.Bytes()
	t.Logf("raw: %v", raw)
	arr.Parse(raw)
	got := make([]int, count, count)
	for i := 0; i < count; i++ {
		got[i] = arr.Pop()
	}

	assert.EqualValuesf(t, expect, got, "bits count: %d", bits)

	assert.EqualValues(t, len(raw), arr.Size(), ".Size()")
}
