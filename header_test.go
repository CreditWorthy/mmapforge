package mmapforge

import (
	"errors"
	"testing"
)

func validHeader() *Header {
	return &Header{
		Magic:         Magic,
		FormatVersion: Version,
		SchemaVersion: 3,
		RecordSize:    72,
		RecordCount:   100,
		Capacity:      100,
	}
}

func TestEncodeHeader_BufferTooSmall(t *testing.T) {
	dst := make([]byte, 32) // less than HeaderSize
	err := EncodeHeader(dst, validHeader())
	if err == nil {
		t.Fatal("expected error for undersized buffer")
	}
}

func TestEncodeHeader_OK(t *testing.T) {
	dst := make([]byte, HeaderSize)
	err := EncodeHeader(dst, validHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// spot-check magic
	if string(dst[0:4]) != "MMFG" {
		t.Fatalf("magic = %q, want MMFG", dst[0:4])
	}
}

func TestDecodeHeader_BufferTooSmall(t *testing.T) {
	_, err := DecodeHeader(make([]byte, 10))
	if err == nil {
		t.Fatal("expected error for undersized buffer")
	}
}

func TestDecodeHeader_BadMagic(t *testing.T) {
	buf := make([]byte, HeaderSize)
	copy(buf[0:4], "NOPE")
	_, err := DecodeHeader(buf)
	if !errors.Is(err, ErrBadMagic) {
		t.Fatalf("err = %v, want ErrBadMagic", err)
	}
}

func TestDecodeHeader_BadVersion(t *testing.T) {
	buf := make([]byte, HeaderSize)
	copy(buf[0:4], "MMFG")
	buf[4] = 99 // bogus version
	_, err := DecodeHeader(buf)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
}

func TestDecodeHeader_RoundTrip(t *testing.T) {
	h := validHeader()
	h.SchemaHash = [32]byte{0xAA, 0xBB} // non-zero to verify copy

	buf := make([]byte, HeaderSize)
	if err := EncodeHeader(buf, h); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := DecodeHeader(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.FormatVersion != h.FormatVersion {
		t.Errorf("FormatVersion = %d, want %d", got.FormatVersion, h.FormatVersion)
	}
	if got.SchemaHash != h.SchemaHash {
		t.Errorf("SchemaHash mismatch")
	}
	if got.SchemaVersion != h.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, h.SchemaVersion)
	}
	if got.RecordSize != h.RecordSize {
		t.Errorf("RecordSize = %d, want %d", got.RecordSize, h.RecordSize)
	}
	if got.RecordCount != h.RecordCount {
		t.Errorf("RecordCount = %d, want %d", got.RecordCount, h.RecordCount)
	}
	if got.Capacity != h.Capacity {
		t.Errorf("Capacity = %d, want %d", got.Capacity, h.Capacity)
	}
}
