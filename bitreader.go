package main

import (
	"errors"
	"io"
)

type BitReader struct {
	data        []byte
	bitPosition uint32
}

func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data, bitPosition: 0}
}

func (br *BitReader) ReadUbit() (uint32, error) {
	ubit, err := br.ReadNBits(6)
	if err != nil {
		return 0, err
	}
	switch ubit & 48 {
	case 16:
		nextBits, err := br.ReadNBits(4)
		if err != nil {
			return 0, err
		}
		return (ubit & 15) | (nextBits << 4), nil
	case 32:
		nextBits, err := br.ReadNBits(8)
		if err != nil {
			return 0, err
		}
		return (ubit & 15) | (nextBits << 4), nil
	case 48:
		nextBits, err := br.ReadNBits(28)
		if err != nil {
			return 0, err
		}
		return (ubit & 15) | (nextBits << 4), nil
	default:
		return ubit, nil
	}
}

func (br *BitReader) ReadVarInt32() (int32, error) {
	var result int32
	var shift uint
	for {
		if shift >= 35 {
			return 0, errors.New("VarInt32 is too long")
		}
		b, err := br.ReadNBits(8)
		if err != nil {
			return 0, err
		}
		result |= int32(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, nil
}

func (br *BitReader) ReadBytes(n int) ([]byte, error) {
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		b, err := br.ReadNBits(8)
		if err != nil {
			return nil, err
		}
		result[i] = byte(b)
	}
	return result, nil
}

func (br *BitReader) ReadNBits(n uint32) (uint32, error) {
	if n > 32 {
		return 0, errors.New("cannot read more than 32 bits")
	}

	result := uint32(0)
	bitsLeft := n

	for bitsLeft > 0 {
		if br.bitPosition/8 >= uint32(len(br.data)) {
			return 0, io.EOF
		}

		byteIndex := br.bitPosition / 8
		bitIndex := br.bitPosition % 8
		bitsAvailable := uint32(8 - bitIndex)
		bitsToRead := bitsLeft
		if bitsToRead > bitsAvailable {
			bitsToRead = bitsAvailable
		}

		mask := uint32((1 << bitsToRead) - 1)
		bits := uint32(br.data[byteIndex]>>bitIndex) & mask
		result |= bits << (n - bitsLeft)

		br.bitPosition += bitsToRead
		bitsLeft -= bitsToRead
	}

	return result, nil
}
