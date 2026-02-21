//go:build unix

package mmapforge

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"
)

// ReadBool reads a bool from record idx at the given byte offset.
func (s *Store) ReadBool(idx int, offset uint32) (bool, error) {
	b, err := s.fieldSlice(idx, offset, 1)
	if err != nil {
		return false, err
	}
	return b[0] == 1, nil
}

// ReadInt8 reads an int8 from record idx at the given byte offset.
func (s *Store) ReadInt8(idx int, offset uint32) (int8, error) {
	b, err := s.fieldSlice(idx, offset, 1)
	if err != nil {
		return 0, err
	}
	return int8(b[0]), nil
}

// ReadUint8 reads a uint8 from record idx at the given byte offset.
func (s *Store) ReadUint8(idx int, offset uint32) (uint8, error) {
	b, err := s.fieldSlice(idx, offset, 1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// ReadInt16 reads an int16 from record idx at the given byte offset.
func (s *Store) ReadInt16(idx int, offset uint32) (int16, error) {
	b, err := s.fieldSlice(idx, offset, 2)
	if err != nil {
		return 0, err
	}
	return *(*int16)(unsafe.Pointer(&b[0])), nil
}

// ReadUint16 reads a uint16 from record idx at the given byte offset.
func (s *Store) ReadUint16(idx int, offset uint32) (uint16, error) {
	b, err := s.fieldSlice(idx, offset, 2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

// ReadInt32 reads an int32 from record idx at the given byte offset.
func (s *Store) ReadInt32(idx int, offset uint32) (int32, error) {
	b, err := s.fieldSlice(idx, offset, 4)
	if err != nil {
		return 0, err
	}
	return *(*int32)(unsafe.Pointer(&b[0])), nil
}

// ReadUint32 reads a uint32 from record idx at the given byte offset.
func (s *Store) ReadUint32(idx int, offset uint32) (uint32, error) {
	b, err := s.fieldSlice(idx, offset, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

// ReadInt64 reads an int64 from record idx at the given byte offset.
func (s *Store) ReadInt64(idx int, offset uint32) (int64, error) {
	b, err := s.fieldSlice(idx, offset, 8)
	if err != nil {
		return 0, err
	}
	return *(*int64)(unsafe.Pointer(&b[0])), nil
}

// ReadUint64 reads a uint64 from record idx at the given byte offset.
func (s *Store) ReadUint64(idx int, offset uint32) (uint64, error) {
	b, err := s.fieldSlice(idx, offset, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

// ReadFloat32 reads a float32 from record idx at the given byte offset.
func (s *Store) ReadFloat32(idx int, offset uint32) (float32, error) {
	b, err := s.fieldSlice(idx, offset, 4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), nil
}

// ReadFloat64 reads a float64 from record idx at the given byte offset.
func (s *Store) ReadFloat64(idx int, offset uint32) (float64, error) {
	b, err := s.fieldSlice(idx, offset, 8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

// ReadString returns a zero-copy string from the mmap region.
// The returned string is valid only until Close() is called.
// fieldSize is the total field size
func (s *Store) ReadString(idx int, offset, fieldSize, maxSize uint32) (string, error) {
	b, err := s.fieldSlice(idx, offset, fieldSize)
	if err != nil {
		return "", err
	}
	strLen := binary.LittleEndian.Uint32(b[:4])
	if strLen > maxSize {
		return "", fmt.Errorf("mmapforge: field at offset %d: %w (len=%d max=%d)", offset, ErrCorrupted, strLen, maxSize)
	}
	if strLen == 0 {
		return "", nil
	}
	data := b[4 : 4+strLen]
	return unsafe.String(&data[0], len(data)), nil
}

// ReadBytes returns a zero-copy byte slice from the mmap region.
// The returned slice is valid only until Close() is called.
func (s *Store) ReadBytes(idx int, offset, fieldSize, maxSize uint32) ([]byte, error) {
	b, err := s.fieldSlice(idx, offset, fieldSize)
	if err != nil {
		return nil, err
	}
	byteLen := binary.LittleEndian.Uint32(b[:4])
	if byteLen > maxSize {
		return nil, fmt.Errorf("mmapforge: field at offset %d: %w (len=%d max=%d)", offset, ErrCorrupted, byteLen, maxSize)
	}
	return b[4 : 4+byteLen], nil
}
