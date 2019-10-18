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

	arr := NewBitArray(bits, uint(len(expect)))
	for _, v := range expect {
		arr.Push(v)
	}

	raw := arr.Bytes()
	got := arr.Parse(raw)

	assert.EqualValuesf(t, expect, got, "bits count: %d", bits)

	assert.EqualValues(t, len(raw), arr.Size(), ".Size()")
}
