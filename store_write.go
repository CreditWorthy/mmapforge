//go:build unix

package mmapforge

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"
)

func (s *Store) WriteBool(idx int, offset uint32, val bool) error {
	b, err := s.fieldSlice(idx, offset, 1)
	if err != nil {
		return err
	}
	if val {
		b[0] = 1
	} else {
		b[0] = 0
	}
	return nil
}

func (s *Store) WriteInt8(idx int, offset uint32, val int8) error {
	b, err := s.fieldSlice(idx, offset, 1)
	if err != nil {
		return err
	}
	b[0] = byte(val)
	return nil
}

func (s *Store) WriteUint8(idx int, offset uint32, val uint8) error {
	b, err := s.fieldSlice(idx, offset, 1)
	if err != nil {
		return err
	}
	b[0] = val
	return nil
}

func (s *Store) WriteInt16(idx int, offset uint32, val int16) error {
	b, err := s.fieldSlice(idx, offset, 2)
	if err != nil {
		return err
	}
	*(*int16)(unsafe.Pointer(&b[0])) = val
	return nil
}

func (s *Store) WriteUint16(idx int, offset uint32, val uint16) error {
	b, err := s.fieldSlice(idx, offset, 2)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(b, val)
	return nil
}

func (s *Store) WriteInt32(idx int, offset uint32, val int32) error {
	b, err := s.fieldSlice(idx, offset, 4)
	if err != nil {
		return err
	}
	*(*int32)(unsafe.Pointer(&b[0])) = val
	return nil
}

func (s *Store) WriteUint32(idx int, offset uint32, val uint32) error {
	b, err := s.fieldSlice(idx, offset, 4)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(b, val)
	return nil
}

func (s *Store) WriteInt64(idx int, offset uint32, val int64) error {
	b, err := s.fieldSlice(idx, offset, 8)
	if err != nil {
		return err
	}
	*(*int64)(unsafe.Pointer(&b[0])) = val
	return nil
}

func (s *Store) WriteUint64(idx int, offset uint32, val uint64) error {
	b, err := s.fieldSlice(idx, offset, 8)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(b, val)
	return nil
}

func (s *Store) WriteFloat32(idx int, offset uint32, val float32) error {
	b, err := s.fieldSlice(idx, offset, 4)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(b, math.Float32bits(val))
	return nil
}

func (s *Store) WriteFloat64(idx int, offset uint32, val float64) error {
	b, err := s.fieldSlice(idx, offset, 8)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(b, math.Float64bits(val))
	return nil
}

// maxLenThreshold is the upper bound for string/byte lengths representable
// by a 4-byte LE prefix. Defaults to math.MaxUint32; tests override it to
// exercise the overflow path without allocating alot here
var maxLenThreshold = math.MaxUint32

// WriteString writes a length-prefixed string into the field, zero-padding the remainder.
func (s *Store) WriteString(idx int, offset, fieldSize, maxSize uint32, val string) error {
	n := len(val)
	if n <= maxLenThreshold {
		if n > int(maxSize) {
			return fmt.Errorf("mmapforge: field at offset %d: %w (len=%d max=%d)", offset, ErrStringTooLong, len(val), maxSize)
		}
		b, err := s.fieldSlice(idx, offset, fieldSize)
		if err != nil {
			return err
		}

		*(*uint32)(unsafe.Pointer(&b[0])) = *(*uint32)(unsafe.Pointer(&n))
		copy(b[4:], val)
		clear(b[4+n:])
		return nil
	}

	return fmt.Errorf("mmapforge: field at offset %d: %w (len=%d max=%d)", offset, ErrStringTooLong, len(val), maxSize)
}

// WriteBytes writes a length-prefixed byte slice into the field, zero-padding the remainder.
func (s *Store) WriteBytes(idx int, offset, fieldSize, maxSize uint32, val []byte) error {
	n := len(val)
	if n >= 0 && n <= maxLenThreshold {
		if n > int(maxSize) {
			return fmt.Errorf("mmapforge: field at offset %d: %w (len=%d max=%d)", offset, ErrBytesTooLong, len(val), maxSize)
		}
		b, err := s.fieldSlice(idx, offset, fieldSize)
		if err != nil {
			return err
		}

		*(*uint32)(unsafe.Pointer(&b[0])) = *(*uint32)(unsafe.Pointer(&n))
		copy(b[4:], val)
		clear(b[4+n:])
		return nil
	}

	return fmt.Errorf("mmapforge: field at offset %d: %w (len=%d max=%d)", offset, ErrBytesTooLong, len(val), maxSize)
}
