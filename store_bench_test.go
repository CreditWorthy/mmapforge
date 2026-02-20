//go:build unix

package mmapforge

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

const benchRecords = 1024

func benchLayout() *RecordLayout {
	layout, layoutErr := ComputeLayout([]FieldDef{
		{Name: "id", Type: FieldUint64},
		{Name: "value", Type: FieldFloat64},
		{Name: "score", Type: FieldInt32},
		{Name: "flags", Type: FieldUint8},
		{Name: "name", Type: FieldString, MaxSize: 32},
	})
	if layoutErr != nil {
		panic(layoutErr)
	}
	return layout
}

func benchStore(b *testing.B) *Store {
	b.Helper()
	path := filepath.Join(b.TempDir(), "bench.mmf")
	s, createErr := CreateStore(path, benchLayout(), 1)
	if createErr != nil {
		b.Fatalf("CreateStore: %v", createErr)
	}
	for i := 0; i < benchRecords; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			b.Fatalf("Append: %v", appendErr)
		}
	}
	return s
}

func benchNameField() FieldLayout {
	layout := benchLayout()
	for _, f := range layout.Fields {
		if f.Name == "name" {
			return f
		}
	}
	panic("name field not found")
}

func fieldOffset(name string) uint32 {
	layout := benchLayout()
	for _, f := range layout.Fields {
		if f.Name == name {
			return f.Offset
		}
	}
	panic("unknown field: " + name)
}

func BenchmarkReadUint64(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("id")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.ReadUint64(i%benchRecords, off); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkReadFloat64(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("value")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.ReadFloat64(i%benchRecords, off); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkReadInt32(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("score")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.ReadInt32(i%benchRecords, off); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkReadUint8(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("flags")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.ReadUint8(i%benchRecords, off); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkReadString(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	nf := benchNameField()
	for i := 0; i < benchRecords; i++ {
		if writeErr := s.WriteString(i, nf.Offset, nf.Size, nf.MaxSize, "hello"); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.ReadString(i%benchRecords, nf.Offset, nf.Size, nf.MaxSize); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkWriteUint64(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("id")
	b.ResetTimer()
	b.ReportAllocs()
	var v uint64
	for i := 0; i < b.N; i++ {
		v++
		if writeErr := s.WriteUint64(i%benchRecords, off, v); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
}

func BenchmarkWriteFloat64(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("value")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if writeErr := s.WriteFloat64(i%benchRecords, off, float64(i)); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
}

func BenchmarkWriteInt32(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	off := fieldOffset("score")
	b.ResetTimer()
	b.ReportAllocs()
	var v int32
	for i := 0; i < b.N; i++ {
		v++
		if writeErr := s.WriteInt32(i%benchRecords, off, v); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
}

func BenchmarkWriteString(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	nf := benchNameField()
	val := "benchmark-test-string"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if writeErr := s.WriteString(i%benchRecords, nf.Offset, nf.Size, nf.MaxSize, val); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
}

func BenchmarkAppend(b *testing.B) {
	path := filepath.Join(b.TempDir(), "bench_append.mmf")
	lay := benchLayout()
	s, createErr := CreateStore(path, lay, 1)
	if createErr != nil {
		b.Fatalf("CreateStore: %v", createErr)
	}
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			b.StopTimer()
			s.Close()
			os.Remove(path)
			s, createErr = CreateStore(path, lay, 1)
			if createErr != nil {
				b.Fatalf("CreateStore: %v", createErr)
			}
			b.StartTimer()
		}
	}
}

func BenchmarkSeqReadBegin(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.SeqReadBegin(i % benchRecords)
	}
}

func BenchmarkSeqWriteCycle(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		idx := i % benchRecords
		s.SeqBeginWrite(idx)
		s.SeqEndWrite(idx)
	}
}

func BenchmarkBaseline_FileReadUint64(b *testing.B) {
	path := filepath.Join(b.TempDir(), "baseline.bin")
	const nRecords = 1024
	const recSize = 24

	f, createErr := os.Create(path)
	if createErr != nil {
		b.Fatal(createErr)
	}
	buf := make([]byte, nRecords*recSize)
	var id64 uint64
	var tag32 uint32
	for i := 0; i < nRecords; i++ {
		binary.LittleEndian.PutUint64(buf[i*recSize:], id64)
		binary.LittleEndian.PutUint64(buf[i*recSize+8:], math.Float64bits(float64(i)))
		binary.LittleEndian.PutUint32(buf[i*recSize+16:], tag32)
		id64++
		tag32++
	}
	if _, writeErr := f.Write(buf); writeErr != nil {
		b.Fatal(writeErr)
	}
	f.Close()

	f, openErr := os.Open(path)
	if openErr != nil {
		b.Fatal(openErr)
	}
	defer f.Close()

	readBuf := make([]byte, 8)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		off := int64((i % nRecords) * recSize)
		if _, readErr := f.ReadAt(readBuf, off); readErr != nil {
			b.Fatal(readErr)
		}
		_ = binary.LittleEndian.Uint64(readBuf)
	}
}

func BenchmarkBaseline_FileWriteUint64(b *testing.B) {
	path := filepath.Join(b.TempDir(), "baseline_w.bin")
	const nRecords = 1024
	const recSize = 24

	f, createErr := os.Create(path)
	if createErr != nil {
		b.Fatal(createErr)
	}
	if truncErr := f.Truncate(nRecords * recSize); truncErr != nil {
		b.Fatal(truncErr)
	}
	defer f.Close()

	writeBuf := make([]byte, 8)
	b.ResetTimer()
	b.ReportAllocs()
	var wv uint64
	for i := 0; i < b.N; i++ {
		off := int64((i % nRecords) * recSize)
		binary.LittleEndian.PutUint64(writeBuf, wv)
		wv++
		if _, writeErr := f.WriteAt(writeBuf, off); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
}

func BenchmarkReadMultiField(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	offID := fieldOffset("id")
	offVal := fieldOffset("value")
	offScore := fieldOffset("score")
	nf := benchNameField()
	var seedU64 uint64
	var seedI32 int32
	for i := 0; i < benchRecords; i++ {
		if writeErr := s.WriteUint64(i, offID, seedU64); writeErr != nil {
			b.Fatal(writeErr)
		}
		if writeErr := s.WriteFloat64(i, offVal, float64(i)); writeErr != nil {
			b.Fatal(writeErr)
		}
		if writeErr := s.WriteInt32(i, offScore, seedI32); writeErr != nil {
			b.Fatal(writeErr)
		}
		seedU64++
		seedI32++
		if writeErr := s.WriteString(i, nf.Offset, nf.Size, nf.MaxSize, "test"); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		idx := i % benchRecords
		if _, readErr := s.ReadUint64(idx, offID); readErr != nil {
			b.Fatal(readErr)
		}
		if _, readErr := s.ReadFloat64(idx, offVal); readErr != nil {
			b.Fatal(readErr)
		}
		if _, readErr := s.ReadInt32(idx, offScore); readErr != nil {
			b.Fatal(readErr)
		}
		if _, readErr := s.ReadString(idx, nf.Offset, nf.Size, nf.MaxSize); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkWriteMultiField(b *testing.B) {
	s := benchStore(b)
	defer s.Close()
	offID := fieldOffset("id")
	offVal := fieldOffset("value")
	offScore := fieldOffset("score")
	nf := benchNameField()
	val := "test"
	b.ResetTimer()
	b.ReportAllocs()
	var wu64 uint64
	var wi32 int32
	for i := 0; i < b.N; i++ {
		idx := i % benchRecords
		if writeErr := s.WriteUint64(idx, offID, wu64); writeErr != nil {
			b.Fatal(writeErr)
		}
		if writeErr := s.WriteFloat64(idx, offVal, float64(i)); writeErr != nil {
			b.Fatal(writeErr)
		}
		if writeErr := s.WriteInt32(idx, offScore, wi32); writeErr != nil {
			b.Fatal(writeErr)
		}
		wu64++
		wi32++
		if writeErr := s.WriteString(idx, nf.Offset, nf.Size, nf.MaxSize, val); writeErr != nil {
			b.Fatal(writeErr)
		}
	}
}
