//go:build unix

package mmapforge

import (
	"testing"
)

func TestWriteBool(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteBool(idx, 8, true); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadBool(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Errorf("WriteBool true: got %v", got)
	}

	if err = s.WriteBool(idx, 8, false); err != nil {
		t.Fatal(err)
	}
	got, err = s.ReadBool(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != false {
		t.Errorf("WriteBool false: got %v", got)
	}
}

func TestWriteBool_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteBool(999, 8, true); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteInt8(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteInt8(idx, 8, -127); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadInt8(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != -127 {
		t.Errorf("WriteInt8 = %d, want -127", got)
	}
}

func TestWriteInt8_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteInt8(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteUint8(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteUint8(idx, 8, 255); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadUint8(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != 255 {
		t.Errorf("WriteUint8 = %d, want 255", got)
	}
}

func TestWriteUint8_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteUint8(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteInt16(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteInt16(idx, 8, -32000); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadInt16(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != -32000 {
		t.Errorf("WriteInt16 = %d, want -32000", got)
	}
}

func TestWriteInt16_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteInt16(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteUint16(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteUint16(idx, 8, 65000); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadUint16(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != 65000 {
		t.Errorf("WriteUint16 = %d, want 65000", got)
	}
}

func TestWriteUint16_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteUint16(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteInt32(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteInt32(idx, 8, -2000000000); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadInt32(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != -2000000000 {
		t.Errorf("WriteInt32 = %d, want -2000000000", got)
	}
}

func TestWriteInt32_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteInt32(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteUint32(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteUint32(idx, 8, 4000000000); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadUint32(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != 4000000000 {
		t.Errorf("WriteUint32 = %d, want 4000000000", got)
	}
}

func TestWriteUint32_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteUint32(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteInt64(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteInt64(idx, 8, -9000000000000); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadInt64(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != -9000000000000 {
		t.Errorf("WriteInt64 = %d, want -9000000000000", got)
	}
}

func TestWriteInt64_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteInt64(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteUint64(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteUint64(idx, 8, 18000000000000000000); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadUint64(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != 18000000000000000000 {
		t.Errorf("WriteUint64 = %d, want 18000000000000000000", got)
	}
}

func TestWriteUint64_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteUint64(0, 8, 1); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteFloat32(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteFloat32(idx, 8, 1.5); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadFloat32(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1.5 {
		t.Errorf("WriteFloat32 = %v, want 1.5", got)
	}
}

func TestWriteFloat32_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteFloat32(0, 8, 1.0); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteFloat64(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteFloat64(idx, 8, 2.718281828); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadFloat64(idx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2.718281828 {
		t.Errorf("WriteFloat64 = %v, want 2.718281828", got)
	}
}

func TestWriteFloat64_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if err := s.WriteFloat64(0, 8, 1.0); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteString(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[0]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	if err = s.WriteString(idx, f.Offset, f.Size, f.MaxSize, "test"); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadString(idx, f.Offset, f.Size, f.MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if got != "test" {
		t.Errorf("WriteString = %q, want %q", got, "test")
	}
}

func TestWriteString_TooLong(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[0]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	long := make([]byte, f.MaxSize+1)
	for i := range long {
		long[i] = 'x'
	}
	if err := s.WriteString(idx, f.Offset, f.Size, f.MaxSize, string(long)); err == nil {
		t.Fatal("expected error for string too long")
	}
}

func TestWriteString_OutOfBounds(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[0]
	if err := s.WriteString(0, f.Offset, f.Size, f.MaxSize, "x"); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteBytes(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[1]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	input := []byte{1, 2, 3, 4}
	if err = s.WriteBytes(idx, f.Offset, f.Size, f.MaxSize, input); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadBytes(idx, f.Offset, f.Size, f.MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(input) {
		t.Fatalf("WriteBytes len = %d, want %d", len(got), len(input))
	}
	for i := range input {
		if got[i] != input[i] {
			t.Errorf("WriteBytes[%d] = %d, want %d", i, got[i], input[i])
		}
	}
}

func TestWriteBytes_TooLong(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[1]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	long := make([]byte, f.MaxSize+1)
	if err := s.WriteBytes(idx, f.Offset, f.Size, f.MaxSize, long); err == nil {
		t.Fatal("expected error for bytes too long")
	}
}

func TestWriteBytes_OutOfBounds(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[1]
	if err := s.WriteBytes(0, f.Offset, f.Size, f.MaxSize, []byte{1}); err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestWriteString_ExceedsLenThreshold(t *testing.T) {
	old := maxLenThreshold
	maxLenThreshold = 2
	defer func() { maxLenThreshold = old }()

	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[0]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	if err := s.WriteString(idx, f.Offset, f.Size, f.MaxSize, "abc"); err == nil {
		t.Fatal("expected error for string exceeding len threshold")
	}
}

func TestWriteBytes_ExceedsLenThreshold(t *testing.T) {
	old := maxLenThreshold
	maxLenThreshold = 2
	defer func() { maxLenThreshold = old }()

	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	f := layout.Fields[1]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	if err := s.WriteBytes(idx, f.Offset, f.Size, f.MaxSize, []byte{1, 2, 3}); err == nil {
		t.Fatal("expected error for bytes exceeding len threshold")
	}
}

func TestWriteBool_Closed(t *testing.T) {
	s := mustCreateStore(t)
	if _, appendErr := s.Append(); appendErr != nil {
		t.Fatal(appendErr)
	}
	s.Close()

	if err := s.WriteBool(0, 8, true); err == nil {
		t.Fatal("expected error writing to closed store")
	}
}
