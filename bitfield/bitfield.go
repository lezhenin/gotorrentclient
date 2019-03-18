package bitfield

import "fmt"

type Bitfield struct {
	field           []byte
	length          uint
	effectiveLength uint
	arrayLength     uint
}

func (b *Bitfield) convertIndex(index uint) (byteIndex, bitIndex uint) {

	if index > b.effectiveLength {
		panic(fmt.Errorf("convert index: index (%d) > effective length (%d)",
			index, b.effectiveLength))
	}

	byteIndex = index / 8
	bitIndex = index % 8

	return byteIndex, bitIndex
}

func (b *Bitfield) Set(index uint) {

	byteIndex, bitIndex := b.convertIndex(index)
	b.field[byteIndex] = b.field[byteIndex] | (0x80 >> bitIndex)
}

func (b *Bitfield) Clear(index uint) {

	byteIndex, bitIndex := b.convertIndex(index)
	b.field[byteIndex] = b.field[byteIndex] &^ (0x80 >> bitIndex)
}

func (b *Bitfield) Get(index uint) (val byte) {

	byteIndex, bitIndex := b.convertIndex(index)
	val = (b.field[byteIndex] << bitIndex) & 0x80

	return val >> 7
}

func (b *Bitfield) Bytes() (val []byte) {
	val = make([]byte, b.arrayLength)
	copy(val, b.field)
	return val
}

func (b *Bitfield) Length() uint {
	return b.effectiveLength
}

func (b *Bitfield) GetIndices(valueAt byte) (indices []uint) {

	indices = []uint{}
	for i := uint(0); i < b.effectiveLength; i++ {
		if b.Get(i) == valueAt {
			indices = append(indices, i)
		}
	}
	return indices
}

func NewBitfield(length uint) (b *Bitfield) {

	b = new(Bitfield)

	b.effectiveLength = length
	b.arrayLength = length / 8
	if length%8 > 0 {
		b.arrayLength += 1
	}
	b.length = b.arrayLength * 8

	b.field = make([]byte, b.arrayLength)

	return b
}

func NewBitfieldFromBytes(field []byte, effectiveLength uint) (b *Bitfield, err error) {

	if len(field)*8 < int(effectiveLength) {
		return nil,
			fmt.Errorf("new bitfield: byte slice len less than field len")
	}

	b = new(Bitfield)

	b.effectiveLength = effectiveLength
	b.arrayLength = uint(len(field))
	b.length = b.arrayLength * 8

	b.field = make([]byte, len(field))
	copy(b.field, field)

	return b, nil
}

func Xor(a *Bitfield, b *Bitfield) (c *Bitfield) {

	checkLength(a, b)

	c = NewBitfield(a.effectiveLength)

	for i := uint(0); i < a.arrayLength; i++ {
		c.field[i] = a.field[i] ^ b.field[i]
	}

	return c
}

func And(a *Bitfield, b *Bitfield) (c *Bitfield) {

	checkLength(a, b)

	c = NewBitfield(a.effectiveLength)

	for i := uint(0); i < a.arrayLength; i++ {
		c.field[i] = a.field[i] & b.field[i]
	}

	return c
}

func Or(a *Bitfield, b *Bitfield) (c *Bitfield) {

	checkLength(a, b)

	c = NewBitfield(a.effectiveLength)

	for i := uint(0); i < a.arrayLength; i++ {
		c.field[i] = a.field[i] | b.field[i]
	}

	return c
}

func AndNot(a *Bitfield, b *Bitfield) (c *Bitfield) {

	checkLength(a, b)

	c = NewBitfield(a.effectiveLength)

	for i := uint(0); i < a.arrayLength; i++ {
		c.field[i] = a.field[i] &^ b.field[i]
	}

	return c
}

func checkLength(a *Bitfield, b *Bitfield) {
	if a.effectiveLength != b.effectiveLength {
		panic("xor: bitfields have different length")
	}
	if a.arrayLength != b.arrayLength {
		panic("xor: underlying arrays have different length")
	}
}
