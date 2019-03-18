package bitfield

import (
	"testing"
)

const bitfieldLength = 896

func TestBitfield_Set(t *testing.T) {

	b := NewBitfield(bitfieldLength)

	if b.arrayLength != bitfieldLength/8 {
		t.Error("Bitfield byte array has wrong length ")
	}

	b.Set(0)
	if b.field[0] != 0x80 {
		t.Error()
	}

	b.Set(5)
	if b.field[0] != 0x84 {
		t.Error()
	}

	b.Set(7)
	if b.field[0] != 0x85 {
		t.Error()
	}

	b.Set(8)
	if b.field[1] != 0x80 {
		t.Error()
	}

	b.Set(125)
	if b.field[15] != 0x04 {
		t.Error()
	}

}

func TestBitfield_Clear(t *testing.T) {

	b := NewBitfield(bitfieldLength)

	if b.arrayLength != bitfieldLength/8 {
		t.Error("Bitfield byte array has wrong length ")
	}

	b.field[0] = 0x85
	b.Clear(7)
	if b.field[0] != 0x84 {
		t.Error()
	}

	b.Clear(5)
	if b.field[0] != 0x80 {
		t.Error()
	}

	b.Clear(0)
	if b.field[0] != 0x00 {
		t.Error()
	}

	b.field[15] = 0x04
	b.Clear(125)
	if b.field[15] != 0x00 {
		t.Error()
	}

}

func TestAnd(t *testing.T) {

	b := NewBitfield(bitfieldLength)
	a := NewBitfield(bitfieldLength)

	a.field[15] = 0x85
	b.field[15] = 0x86

	c := And(a, b)

	if c.field[15] != 0x85&0x86 {
		t.Error()
	}

}

func BenchmarkAnd(b *testing.B) {

	d := NewBitfield(1024 * 1024)
	a := NewBitfield(1024 * 1024)

	for i := 0; i < b.N; i++ {
		_ = And(a, d)
	}
}
