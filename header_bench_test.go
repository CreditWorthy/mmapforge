package mmapforge

import (
	"testing"
)

func BenchmarkEncodeHeader(b *testing.B) {
	h := &Header{
		Magic:         Magic,
		FormatVersion: Version,
		SchemaHash:    [32]byte{0xAA, 0xBB, 0xCC},
		SchemaVersion: 3,
		RecordSize:    72,
		RecordCount:   10000,
		Capacity:      16384,
	}
	dst := make([]byte, HeaderSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if encodeErr := EncodeHeader(dst, h); encodeErr != nil {
			b.Fatal(encodeErr)
		}
	}
}

func BenchmarkDecodeHeader(b *testing.B) {
	h := &Header{
		Magic:         Magic,
		FormatVersion: Version,
		SchemaHash:    [32]byte{0xAA, 0xBB, 0xCC},
		SchemaVersion: 3,
		RecordSize:    72,
		RecordCount:   10000,
		Capacity:      16384,
	}
	buf := make([]byte, HeaderSize)
	if encodeErr := EncodeHeader(buf, h); encodeErr != nil {
		b.Fatal(encodeErr)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, decodeErr := DecodeHeader(buf); decodeErr != nil {
			b.Fatal(decodeErr)
		}
	}
}
