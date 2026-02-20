//go:build unix

package mmapforge

import (
	"encoding/binary"
	"testing"
)

func TestReadBool(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteBool(idx, 8, true); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadBool(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != true {
		t.Errorf("ReadBool = %v, want true", got)
	}

	if writeErr := s.WriteBool(idx, 8, false); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr = s.ReadBool(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != false {
		t.Errorf("ReadBool = %v, want false", got)
	}
}

func TestReadBool_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadBool(999, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadInt8(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteInt8(idx, 8, -42); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadInt8(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != -42 {
		t.Errorf("ReadInt8 = %d, want -42", got)
	}
}

func TestReadInt8_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadInt8(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadUint8(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteUint8(idx, 8, 200); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadUint8(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != 200 {
		t.Errorf("ReadUint8 = %d, want 200", got)
	}
}

func TestReadUint8_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadUint8(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadInt16(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteInt16(idx, 8, -1234); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadInt16(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != -1234 {
		t.Errorf("ReadInt16 = %d, want -1234", got)
	}
}

func TestReadInt16_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadInt16(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadUint16(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteUint16(idx, 8, 60000); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadUint16(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != 60000 {
		t.Errorf("ReadUint16 = %d, want 60000", got)
	}
}

func TestReadUint16_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadUint16(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadInt32(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteInt32(idx, 8, -100000); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadInt32(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != -100000 {
		t.Errorf("ReadInt32 = %d, want -100000", got)
	}
}

func TestReadInt32_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadInt32(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadUint32(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteUint32(idx, 8, 3000000000); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadUint32(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != 3000000000 {
		t.Errorf("ReadUint32 = %d, want 3000000000", got)
	}
}

func TestReadUint32_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadUint32(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadInt64(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteInt64(idx, 8, -9000000000); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadInt64(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != -9000000000 {
		t.Errorf("ReadInt64 = %d, want -9000000000", got)
	}
}

func TestReadInt64_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadInt64(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadUint64(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := s.WriteUint64(idx, 8, 18000000000000000000); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadUint64(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != 18000000000000000000 {
		t.Errorf("ReadUint64 = %d, want 18000000000000000000", got)
	}
}

func TestReadUint64_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadUint64(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadFloat32(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteFloat32(idx, 8, 3.14); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadFloat32(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != 3.14 {
		t.Errorf("ReadFloat32 = %v, want 3.14", got)
	}
}

func TestReadFloat32_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadFloat32(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadFloat64(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteFloat64(idx, 8, 2.718281828); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, readErr := s.ReadFloat64(idx, 8)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got != 2.718281828 {
		t.Errorf("ReadFloat64 = %v, want 2.718281828", got)
	}
}

func TestReadFloat64_OutOfBounds(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	_, err := s.ReadFloat64(0, 8)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func testStringLayout() *RecordLayout {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "name", GoName: "Name", Type: FieldString, MaxSize: 32},
		{Name: "data", GoName: "Data", Type: FieldBytes, MaxSize: 64},
	})
	if err != nil {
		panic(err)
	}
	return layout
}

func TestReadString(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	nameField := layout.Fields[0]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	if err = s.WriteString(idx, nameField.Offset, nameField.Size, nameField.MaxSize, "hello"); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadString(idx, nameField.Offset, nameField.Size, nameField.MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("ReadString = %q, want %q", got, "hello")
	}
}

func TestReadString_Empty(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	nameField := layout.Fields[0]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadString(idx, nameField.Offset, nameField.Size, nameField.MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("ReadString empty = %q, want empty", got)
	}
}

func TestReadString_Corrupted(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	nameField := layout.Fields[0]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	off := HeaderSize + idx*int(layout.RecordSize) + int(nameField.Offset)
	b := s.region.Slice(off, 4)
	binary.LittleEndian.PutUint32(b, nameField.MaxSize+1)

	_, err = s.ReadString(idx, nameField.Offset, nameField.Size, nameField.MaxSize)
	if err == nil {
		t.Fatal("expected error for corrupted string length")
	}
}

func TestReadString_OutOfBounds(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	nameField := layout.Fields[0]
	_, err = s.ReadString(0, nameField.Offset, nameField.Size, nameField.MaxSize)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadBytes(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	dataField := layout.Fields[1]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	input := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if err = s.WriteBytes(idx, dataField.Offset, dataField.Size, dataField.MaxSize, input); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadBytes(idx, dataField.Offset, dataField.Size, dataField.MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(input) {
		t.Fatalf("ReadBytes len = %d, want %d", len(got), len(input))
	}
	for i := range input {
		if got[i] != input[i] {
			t.Errorf("ReadBytes[%d] = %x, want %x", i, got[i], input[i])
		}
	}
}

func TestReadBytes_Empty(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	dataField := layout.Fields[1]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.ReadBytes(idx, dataField.Offset, dataField.Size, dataField.MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("ReadBytes empty len = %d, want 0", len(got))
	}
}

func TestReadBytes_Corrupted(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	dataField := layout.Fields[1]
	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	off := HeaderSize + idx*int(layout.RecordSize) + int(dataField.Offset)
	b := s.region.Slice(off, 4)
	binary.LittleEndian.PutUint32(b, dataField.MaxSize+1)

	_, err = s.ReadBytes(idx, dataField.Offset, dataField.Size, dataField.MaxSize)
	if err == nil {
		t.Fatal("expected error for corrupted bytes length")
	}
}

func TestReadBytes_OutOfBounds(t *testing.T) {
	layout := testStringLayout()
	path := tempPath(t)
	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	dataField := layout.Fields[1]
	_, err = s.ReadBytes(0, dataField.Offset, dataField.Size, dataField.MaxSize)
	if err == nil {
		t.Fatal("expected error for out of bounds")
	}
}

func TestReadBool_Closed(t *testing.T) {
	s := mustCreateStore(t)
	if _, appendErr := s.Append(); appendErr != nil {
		t.Fatal(appendErr)
	}
	s.Close()

	_, err := s.ReadBool(0, 8)
	if err == nil {
		t.Fatal("expected error reading from closed store")
	}
}
