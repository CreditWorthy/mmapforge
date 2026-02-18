package mmapforge

import (
	"math"
	"testing"
)

func TestComputeLayout_SingleFixedField(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "x", GoName: "X", Type: FieldInt32},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(layout.Fields))
	}
	f := layout.Fields[0]
	if f.Offset != SeqFieldSize {
		t.Errorf("expected offset %d, got %d", SeqFieldSize, f.Offset)
	}
	if f.Size != 4 {
		t.Errorf("expected size 4, got %d", f.Size)
	}
	if f.Align != 4 {
		t.Errorf("expected align 4, got %d", f.Align)
	}
	if layout.RecordSize != 16 {
		t.Errorf("expected record size 16, got %d", layout.RecordSize)
	}
}

func TestComputeLayout_AllFixedTypes(t *testing.T) {
	fields := []FieldDef{
		{Name: "a", GoName: "A", Type: FieldBool},
		{Name: "b", GoName: "B", Type: FieldInt8},
		{Name: "c", GoName: "C", Type: FieldUint8},
		{Name: "d", GoName: "D", Type: FieldInt16},
		{Name: "e", GoName: "E", Type: FieldUint16},
		{Name: "f", GoName: "F", Type: FieldInt32},
		{Name: "g", GoName: "G", Type: FieldUint32},
		{Name: "h", GoName: "H", Type: FieldFloat32},
		{Name: "i", GoName: "I", Type: FieldInt64},
		{Name: "j", GoName: "J", Type: FieldUint64},
		{Name: "k", GoName: "K", Type: FieldFloat64},
	}
	layout, err := ComputeLayout(fields)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Fields) != len(fields) {
		t.Fatalf("expected %d fields, got %d", len(fields), len(layout.Fields))
	}

	for _, f := range layout.Fields {
		if f.Offset%f.Align != 0 {
			t.Errorf("field %q: offset %d not aligned to %d", f.Name, f.Offset, f.Align)
		}
	}

	for i := 0; i < len(layout.Fields)-1; i++ {
		end := layout.Fields[i].Offset + layout.Fields[i].Size
		next := layout.Fields[i+1].Offset
		if end > next {
			t.Errorf("field %q (end=%d) overlaps field %q (start=%d)",
				layout.Fields[i].Name, end, layout.Fields[i+1].Name, next)
		}
	}

	if layout.RecordSize%8 != 0 {
		t.Errorf("record size %d not a multiple of 8", layout.RecordSize)
	}
}

func TestComputeLayout_StringField(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "s", GoName: "S", Type: FieldString, MaxSize: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := layout.Fields[0]
	if f.Size != 104 {
		t.Errorf("expected size 104, got %d", f.Size)
	}
	if f.Align != 4 {
		t.Errorf("expected align 4, got %d", f.Align)
	}
}

func TestComputeLayout_BytesField(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "b", GoName: "B", Type: FieldBytes, MaxSize: 256},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := layout.Fields[0]
	if f.Size != 260 {
		t.Errorf("expected size 260, got %d", f.Size)
	}
}

func TestComputeLayout_AlignmentPadding(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "flag", GoName: "Flag", Type: FieldBool},
		{Name: "val", GoName: "Val", Type: FieldInt64},
	})
	if err != nil {
		t.Fatal(err)
	}
	flag := layout.Fields[0]
	val := layout.Fields[1]

	if flag.Offset != 8 {
		t.Errorf("flag offset: expected 8, got %d", flag.Offset)
	}
	if val.Offset != 16 {
		t.Errorf("val offset: expected 16, got %d", val.Offset)
	}
}

func TestComputeLayout_RecordSizePaddedTo8(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "b", GoName: "B", Type: FieldBool},
	})
	if err != nil {
		t.Fatal(err)
	}
	if layout.RecordSize != 16 {
		t.Errorf("expected record size 16, got %d", layout.RecordSize)
	}
}

func TestComputeLayout_ExactMultipleOf8NoExtraPadding(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "v", GoName: "V", Type: FieldInt64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if layout.RecordSize != 16 {
		t.Errorf("expected record size 16, got %d", layout.RecordSize)
	}
}

func TestComputeLayout_NoFields(t *testing.T) {
	_, err := ComputeLayout(nil)
	if err == nil {
		t.Fatal("expected error for nil fields")
	}
	_, err = ComputeLayout([]FieldDef{})
	if err == nil {
		t.Fatal("expected error for empty fields")
	}
}

func TestComputeLayout_DuplicateName(t *testing.T) {
	_, err := ComputeLayout([]FieldDef{
		{Name: "a", GoName: "A", Type: FieldInt32},
		{Name: "a", GoName: "A2", Type: FieldInt64},
	})
	if err == nil {
		t.Fatal("expected error for duplicate field name")
	}
}

func TestComputeLayout_StringZeroMaxSize(t *testing.T) {
	_, err := ComputeLayout([]FieldDef{
		{Name: "s", GoName: "S", Type: FieldString, MaxSize: 0},
	})
	if err == nil {
		t.Fatal("expected error for string with max_size 0")
	}
}

func TestComputeLayout_BytesZeroMaxSize(t *testing.T) {
	_, err := ComputeLayout([]FieldDef{
		{Name: "b", GoName: "B", Type: FieldBytes, MaxSize: 0},
	})
	if err == nil {
		t.Fatal("expected error for bytes with max_size 0")
	}
}

func TestComputeLayout_MaxSizeOverflow(t *testing.T) {
	_, err := ComputeLayout([]FieldDef{
		{Name: "s", GoName: "S", Type: FieldString, MaxSize: math.MaxUint32},
	})
	if err == nil {
		t.Fatal("expected error for overflowing max_size")
	}
}

func TestComputeLayout_UnknownFieldType(t *testing.T) {
	_, err := ComputeLayout([]FieldDef{
		{Name: "x", GoName: "X", Type: FieldType(999)},
	})
	if err == nil {
		t.Fatal("expected error for unknown field type")
	}
}

func TestFieldType_String(t *testing.T) {
	cases := []struct {
		ft   FieldType
		want string
	}{
		{FieldBool, "bool"},
		{FieldInt8, "int8"},
		{FieldUint8, "uint8"},
		{FieldInt16, "int16"},
		{FieldUint16, "uint16"},
		{FieldInt32, "int32"},
		{FieldUint32, "uint32"},
		{FieldInt64, "int64"},
		{FieldUint64, "uint64"},
		{FieldFloat32, "float32"},
		{FieldFloat64, "float64"},
		{FieldString, "string"},
		{FieldBytes, "bytes"},
		{FieldType(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.ft.String(); got != tc.want {
			t.Errorf("FieldType(%d).String() = %q, want %q", tc.ft, got, tc.want)
		}
	}
}

func TestDescriptors(t *testing.T) {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "id", GoName: "ID", Type: FieldUint64},
		{Name: "name", GoName: "Name", Type: FieldString, MaxSize: 32},
	})
	if err != nil {
		t.Fatal(err)
	}
	descs := layout.Descriptors()
	if len(descs) != 2 {
		t.Fatalf("expected 2 descriptors, got %d", len(descs))
	}
	if descs[0].Name != "id" || descs[0].Type != "uint64" || descs[0].Size != 8 {
		t.Errorf("unexpected descriptor[0]: %+v", descs[0])
	}
	if descs[1].Name != "name" || descs[1].Type != "string" || descs[1].Size != 36 {
		t.Errorf("unexpected descriptor[1]: %+v", descs[1])
	}
}

func TestSchemaHash_Deterministic(t *testing.T) {
	descs := []FieldDescriptor{
		{Name: "x", Type: "int32", Size: 4},
		{Name: "y", Type: "float64", Size: 8},
	}
	h1 := SchemaHash(descs)
	h2 := SchemaHash(descs)
	if h1 != h2 {
		t.Error("same input produced different hashes")
	}
}

func TestSchemaHash_OrderIndependent(t *testing.T) {
	a := []FieldDescriptor{
		{Name: "x", Type: "int32", Size: 4},
		{Name: "y", Type: "float64", Size: 8},
	}
	b := []FieldDescriptor{
		{Name: "y", Type: "float64", Size: 8},
		{Name: "x", Type: "int32", Size: 4},
	}
	if SchemaHash(a) != SchemaHash(b) {
		t.Error("hash should be independent of field order")
	}
}

func TestSchemaHash_DifferentSchemasDiffer(t *testing.T) {
	a := []FieldDescriptor{
		{Name: "x", Type: "int32", Size: 4},
	}
	b := []FieldDescriptor{
		{Name: "x", Type: "int64", Size: 8},
	}
	if SchemaHash(a) == SchemaHash(b) {
		t.Error("different schemas should produce different hashes")
	}
}

func TestSchemaHash_DoesNotMutateInput(t *testing.T) {
	descs := []FieldDescriptor{
		{Name: "b", Type: "int32", Size: 4},
		{Name: "a", Type: "float64", Size: 8},
	}
	SchemaHash(descs)
	if descs[0].Name != "b" || descs[1].Name != "a" {
		t.Error("SchemaHash mutated the input slice")
	}
}

func TestRoundTrip_LayoutToHash(t *testing.T) {
	fields := []FieldDef{
		{Name: "id", GoName: "ID", Type: FieldUint64},
		{Name: "price", GoName: "Price", Type: FieldFloat64},
		{Name: "qty", GoName: "Qty", Type: FieldInt32},
		{Name: "symbol", GoName: "Symbol", Type: FieldString, MaxSize: 16},
		{Name: "active", GoName: "Active", Type: FieldBool},
	}
	layout, err := ComputeLayout(fields)
	if err != nil {
		t.Fatal(err)
	}

	descs := layout.Descriptors()
	h := SchemaHash(descs)

	var zero [32]byte
	if h == zero {
		t.Error("hash should not be all zeros")
	}

	if SchemaHash(layout.Descriptors()) != h {
		t.Error("hash not stable across calls")
	}
}
