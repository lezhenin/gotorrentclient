package bitfield

import (
	"fmt"
	"testing"
)

const bitfieldLength = 30
const bitfieldArrayLength = 4

func TestBitfieldCreation(t *testing.T) {

	const testIndex = 29

	b := NewBitfield(bitfieldLength)

	if b.arrayLength != 4 {
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
