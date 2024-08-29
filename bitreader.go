package main

import (
	"errors"
)

type BitReader struct {
	data        []byte
	bitPosition uint32
}

func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data, bitPosition: 0}
}

func (br *BitReader) ReadNBits(n uint32) (uint32, error) {
	if n > 32 {
		return 0, errors.New("cannot read more than 32 bits")
	}

	result := uint32(0)
	for i := uint32(0); i < n; i++ {
		if br.bitPosition/8 >= uint32(len(br.data)) {
			return 0, errors.New("end of data reached")
		}
		byteIndex := br.bitPosition / 8
		bitIndex := br.bitPosition % 8
		bit := (br.data[byteIndex] >> bitIndex) & 1
		result |= uint32(bit) << i
		br.bitPosition++
	}

	return result, nil
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

func (br *BitReader) ReadVarInt32() (uint32, error) {
	result := uint32(0)
	count := uint32(0)
	for {
		if count >= 5 {
			return 0, errors.New("VarInt32 exceeds 5 bytes")
		}
		b, err := br.ReadNBits(8)
		if err != nil {
			return 0, err
		}
		result |= (b & 0x7F) << (7 * count)
		count++
		if b&0x80 == 0 {
			break
		}
	}
	return result, nil
}

func (br *BitReader) ReadBytes(n int) ([]byte, error) {
	if br.bitPosition/8+uint32(n) > uint32(len(br.data)) {
		return nil, errors.New("not enough data to read")
	}
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
