package bitfield

import "fmt"

type Bitfield struct {
	Field           []byte
	Length          uint
	EffectiveLength uint
	ArrayLength     uint
}

func NewBitfield(Length uint) (b *Bitfield) {

	b = new(Bitfield)

	b.EffectiveLength = Length
	b.ArrayLength = Length / 8
	if Length%8 > 0 {
		b.ArrayLength += 1
	}
	b.Length = b.ArrayLength * 8

	b.Field = make([]byte, b.ArrayLength)

	return b
}

func NewBitfieldFromBytes(field []byte, effectiveLength uint) (b *Bitfield, err error) {

	if len(field)*8 < int(effectiveLength) {
		return nil,
			fmt.Errorf("new butField: byte slice len less than Field len")
	}

	b = new(Bitfield)

	b.EffectiveLength = effectiveLength
	b.ArrayLength = uint(len(field))
	b.Length = b.ArrayLength * 8

	b.Field = make([]byte, len(field))
	copy(b.Field, field)

	return b, nil
}

func (b *Bitfield) Set(index uint) {

	byteIndex, bitIndex := b.convertIndex(index)
	b.Field[byteIndex] = b.Field[byteIndex] | (0x01 << bitIndex)
}

func (b *Bitfield) Clear(index uint) {

	byteIndex, bitIndex := b.convertIndex(index)
	b.Field[byteIndex] = b.Field[byteIndex] &^ (0x01 << bitIndex)
}

func (b *Bitfield) Get(index uint) (val byte) {

	byteIndex, bitIndex := b.convertIndex(index)
	val = (b.Field[byteIndex] >> bitIndex) & 0x01

	return val
}

func (b *Bitfield) Bytes() (val []byte) {
	val = make([]byte, b.ArrayLength)
	copy(val, b.Field)
	return val
}

func (b *Bitfield) convertIndex(index uint) (byteIndex, bitIndex uint) {

	byteIndex = index / 8
	bitIndex = 8 - (index % 8) - 1

	return byteIndex, bitIndex
}
