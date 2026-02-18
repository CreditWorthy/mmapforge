package mmapforge

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Header struct {
	Magic         [4]byte
	FormatVersion uint32
	SchemaHash    [32]byte
	SchemaVersion uint32
	RecordSize    uint32
	RecordCount   uint64
	Capacity      uint64
}

// EncodeHeader writes h into the first 64 bytes of dst.
func EncodeHeader(dst []byte, h *Header) error {
	if len(dst) < HeaderSize {
		return fmt.Errorf("mmapforge: header encode: buffer too small (%d < %d)", len(dst), HeaderSize)
	}
	copy(dst[0:4], Magic[:])
	binary.LittleEndian.PutUint32(dst[4:8], h.FormatVersion)
	copy(dst[8:40], h.SchemaHash[:])
	binary.LittleEndian.PutUint32(dst[40:44], h.SchemaVersion)
	binary.LittleEndian.PutUint32(dst[44:48], h.RecordSize)
	binary.LittleEndian.PutUint64(dst[48:56], h.RecordCount)
	binary.LittleEndian.PutUint64(dst[56:64], h.Capacity)
	return nil
}

// DecodeHeader reads the first 64 bytes of src into a Header.
func DecodeHeader(src []byte) (*Header, error) {
	if len(src) < HeaderSize {
		return nil, fmt.Errorf("mmapforge: header decode: buffer too small (%d < %d)", len(src), HeaderSize)
	}
	h := &Header{}
	if !bytes.Equal(src[0:4], Magic[:]) {
		return nil, fmt.Errorf("mmapforge: header decode: %w (got %q)", ErrBadMagic, src[0:4])
	}
	h.FormatVersion = binary.LittleEndian.Uint32(src[4:8])
	if h.FormatVersion != Version {
		return nil, fmt.Errorf("mmapforge: header decode: unsupported format version %d", h.FormatVersion)
	}
	copy(h.SchemaHash[:], src[8:40])
	h.SchemaVersion = binary.LittleEndian.Uint32(src[40:44])
	h.RecordSize = binary.LittleEndian.Uint32(src[44:48])
	h.RecordCount = binary.LittleEndian.Uint64(src[48:56])
	h.Capacity = binary.LittleEndian.Uint64(src[56:64])
	return h, nil
}
