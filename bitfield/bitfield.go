package bitfield

import "fmt"

type Bitfield struct {
	field           []byte
	length          uint
	effectiveLength uint
	arrayLength     uint
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
			fmt.Errorf("new butField: byte slice len less than field len")
	}

	b = new(Bitfield)

	b.effectiveLength = effectiveLength
	b.arrayLength = uint(len(field))
	b.length = b.arrayLength * 8

	b.field = make([]byte, len(field))
	copy(b.field, field)

	return b, nil
}

func (b *Bitfield) Set(index uint) {

	byteIndex, bitIndex := b.convertIndex(index)
	b.field[byteIndex] = b.field[byteIndex] | (0x01 << bitIndex)
}

func (b *Bitfield) Clear(index uint) {

	byteIndex, bitIndex := b.convertIndex(index)
	b.field[byteIndex] = b.field[byteIndex] &^ (0x01 << bitIndex)
}

func (b *Bitfield) Get(index uint) (val byte) {

	byteIndex, bitIndex := b.convertIndex(index)
	val = (b.field[byteIndex] >> bitIndex) & 0x01

	return val
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

func (b *Bitfield) convertIndex(index uint) (byteIndex, bitIndex uint) {

	byteIndex = index / 8
	bitIndex = 8 - (index % 8) - 1

	return byteIndex, bitIndex
}
