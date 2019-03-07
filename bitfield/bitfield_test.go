package bitfield

import (
	"testing"
)

const bitfieldLength = 896
const bitfieldArrayLength = 4

func TestBitfieldCreation(t *testing.T) {

	const testIndex = 764

	b := NewBitfield(bitfieldLength)

	if b.arrayLength != 112 {
		t.Error("Bitfield byte array has wrong length ")
	}

	if b.Get(testIndex) != 0 {
		t.Error("Value at", testIndex, "is not zero after init")
	}

	b.Set(testIndex)
	if b.Get(testIndex) != 1 {
		t.Error("Value at", testIndex, "is not one after set")
	}

	b.Clear(testIndex)
	if b.Get(testIndex) != 0 {
		t.Error("Value at", testIndex, "is not zero after clear")

	}

}
