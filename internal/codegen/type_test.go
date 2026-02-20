package codegen

import (
	"fmt"
	"testing"

	"github.com/CreditWorthy/mmapforge"
)

func allFieldTypes() []*Field {
	return []*Field{
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "b", GoName: "B", Type: mmapforge.FieldBool}, Offset: 8, Size: 1, Align: 1}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "i8", GoName: "I8", Type: mmapforge.FieldInt8}, Offset: 9, Size: 1, Align: 1}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "u8", GoName: "U8", Type: mmapforge.FieldUint8}, Offset: 10, Size: 1, Align: 1}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "i16", GoName: "I16", Type: mmapforge.FieldInt16}, Offset: 12, Size: 2, Align: 2}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "u16", GoName: "U16", Type: mmapforge.FieldUint16}, Offset: 14, Size: 2, Align: 2}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "i32", GoName: "I32", Type: mmapforge.FieldInt32}, Offset: 16, Size: 4, Align: 4}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "u32", GoName: "U32", Type: mmapforge.FieldUint32}, Offset: 20, Size: 4, Align: 4}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "i64", GoName: "I64", Type: mmapforge.FieldInt64}, Offset: 24, Size: 8, Align: 8}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "u64", GoName: "U64", Type: mmapforge.FieldUint64}, Offset: 32, Size: 8, Align: 8}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "f32", GoName: "F32", Type: mmapforge.FieldFloat32}, Offset: 40, Size: 4, Align: 4}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "f64", GoName: "F64", Type: mmapforge.FieldFloat64}, Offset: 48, Size: 8, Align: 8}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "s", GoName: "S", Type: mmapforge.FieldString, MaxSize: 32}, Offset: 56, Size: 36, Align: 4}},
		{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Name: "bs", GoName: "Bs", Type: mmapforge.FieldBytes, MaxSize: 64}, Offset: 92, Size: 68, Align: 4}},
	}
}

func newType(cfg *Config, name string, fields []*Field) *Type {
	return &Type{
		Config:  cfg,
		Name:    name,
		Package: "main",
		Fields:  fields,
	}
}

func TestType_Header_Default(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.Header(); got != DefaultHeader {
		t.Errorf("Header() = %q, want %q", got, DefaultHeader)
	}
}

func TestType_Header_Custom(t *testing.T) {
	custom := "// custom header"
	ty := newType(&Config{Header: custom}, "Player", nil)
	if got := ty.Header(); got != custom {
		t.Errorf("Header() = %q, want %q", got, custom)
	}
}

func TestType_Label(t *testing.T) {
	ty := newType(&Config{}, "PlayerState", nil)
	if got := ty.Label(); got != "playerstate" {
		t.Errorf("Label() = %q, want %q", got, "playerstate")
	}
}

func TestType_StoreName(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.StoreName(); got != "PlayerStore" {
		t.Errorf("StoreName() = %q, want %q", got, "PlayerStore")
	}
}

func TestType_RecordName(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.RecordName(); got != "PlayerRecord" {
		t.Errorf("RecordName() = %q, want %q", got, "PlayerRecord")
	}
}

func TestType_LayoutFuncName(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.LayoutFuncName(); got != "PlayerLayout" {
		t.Errorf("LayoutFuncName() = %q, want %q", got, "PlayerLayout")
	}
}

func TestType_NewStoreFuncName(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.NewStoreFuncName(); got != "NewPlayerStore" {
		t.Errorf("NewStoreFuncName() = %q, want %q", got, "NewPlayerStore")
	}
}

func TestType_OpenStoreFuncName(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.OpenStoreFuncName(); got != "OpenPlayerStore" {
		t.Errorf("OpenStoreFuncName() = %q, want %q", got, "OpenPlayerStore")
	}
}

func TestType_Receiver(t *testing.T) {
	ty := newType(&Config{}, "Player", nil)
	if got := ty.Receiver(); got != "s" {
		t.Errorf("Receiver() = %q, want %q", got, "s")
	}
}

func TestType_HasStringField(t *testing.T) {
	strField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString, MaxSize: 32}}}
	intField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}}}
	bytField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes, MaxSize: 16}}}

	cases := []struct {
		name   string
		fields []*Field
		want   bool
	}{
		{"nil fields", nil, false},
		{"no string", []*Field{intField}, false},
		{"has string", []*Field{intField, strField}, true},
		{"bytes only", []*Field{bytField}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ty := newType(&Config{}, "X", tc.fields)
			if got := ty.HasStringField(); got != tc.want {
				t.Errorf("HasStringField() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestType_HasBytesField(t *testing.T) {
	strField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString, MaxSize: 32}}}
	intField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}}}
	bytField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes, MaxSize: 16}}}

	cases := []struct {
		name   string
		fields []*Field
		want   bool
	}{
		{"nil fields", nil, false},
		{"no bytes", []*Field{intField}, false},
		{"has bytes", []*Field{intField, bytField}, true},
		{"string only", []*Field{strField}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ty := newType(&Config{}, "X", tc.fields)
			if got := ty.HasBytesField(); got != tc.want {
				t.Errorf("HasBytesField() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestType_HasVarLenField(t *testing.T) {
	strField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString, MaxSize: 32}}}
	intField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}}}
	bytField := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes, MaxSize: 16}}}

	cases := []struct {
		name   string
		fields []*Field
		want   bool
	}{
		{"no varlen", []*Field{intField}, false},
		{"string only", []*Field{strField}, true},
		{"bytes only", []*Field{bytField}, true},
		{"both", []*Field{strField, bytField}, true},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ty := newType(&Config{}, "X", tc.fields)
			if got := ty.HasVarLenField(); got != tc.want {
				t.Errorf("HasVarLenField() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestField_GoType(t *testing.T) {
	want := []string{
		"bool", "int8", "uint8", "int16", "uint16",
		"int32", "uint32", "int64", "uint64",
		"float32", "float64", "string", "[]byte",
	}
	fields := allFieldTypes()
	for i, f := range fields {
		if got := f.GoType(); got != want[i] {
			t.Errorf("GoType() for %s = %q, want %q", f.Name, got, want[i])
		}
	}
}

func TestField_GoType_Unknown(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldType(99)}}}
	if got := f.GoType(); got != "unknown" {
		t.Errorf("GoType() unknown = %q, want %q", got, "unknown")
	}
}

func TestField_GetterName(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{GoName: "Health"}}}
	if got := f.GetterName(); got != "GetHealth" {
		t.Errorf("GetterName() = %q, want %q", got, "GetHealth")
	}
}

func TestField_SetterName(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{GoName: "Health"}}}
	if got := f.SetterName(); got != "SetHealth" {
		t.Errorf("SetterName() = %q, want %q", got, "SetHealth")
	}
}

func TestField_IsString(t *testing.T) {
	yes := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString}}}
	no := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}}}
	if !yes.IsString() {
		t.Error("IsString() should be true for FieldString")
	}
	if no.IsString() {
		t.Error("IsString() should be false for FieldInt32")
	}
}

func TestField_IsBytes(t *testing.T) {
	yes := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes}}}
	no := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}}}
	if !yes.IsBytes() {
		t.Error("IsBytes() should be true for FieldBytes")
	}
	if no.IsBytes() {
		t.Error("IsBytes() should be false for FieldInt32")
	}
}

func TestField_IsVarLen(t *testing.T) {
	str := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString}}}
	byt := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes}}}
	num := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldFloat64}}}
	if !str.IsVarLen() {
		t.Error("IsVarLen() should be true for FieldString")
	}
	if !byt.IsVarLen() {
		t.Error("IsVarLen() should be true for FieldBytes")
	}
	if num.IsVarLen() {
		t.Error("IsVarLen() should be false for FieldFloat64")
	}
}

func TestField_IsNumeric(t *testing.T) {
	numericTypes := []mmapforge.FieldType{
		mmapforge.FieldInt8, mmapforge.FieldUint8,
		mmapforge.FieldInt16, mmapforge.FieldUint16,
		mmapforge.FieldInt32, mmapforge.FieldUint32,
		mmapforge.FieldInt64, mmapforge.FieldUint64,
		mmapforge.FieldFloat32, mmapforge.FieldFloat64,
	}
	for _, ft := range numericTypes {
		f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: ft}}}
		if !f.IsNumeric() {
			t.Errorf("IsNumeric() should be true for %v", ft)
		}
	}

	nonNumeric := []mmapforge.FieldType{
		mmapforge.FieldBool, mmapforge.FieldString, mmapforge.FieldBytes, mmapforge.FieldType(99),
	}
	for _, ft := range nonNumeric {
		f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: ft}}}
		if f.IsNumeric() {
			t.Errorf("IsNumeric() should be false for %v", ft)
		}
	}
}

func TestField_IsBool(t *testing.T) {
	yes := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBool}}}
	no := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}}}
	if !yes.IsBool() {
		t.Error("IsBool() should be true for FieldBool")
	}
	if no.IsBool() {
		t.Error("IsBool() should be false for FieldInt32")
	}
}

func TestField_TypeConstant(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldFloat64}}}
	if got := f.TypeConstant(); got != int(mmapforge.FieldFloat64) {
		t.Errorf("TypeConstant() = %d, want %d", got, int(mmapforge.FieldFloat64))
	}
}

func TestField_ReadCall(t *testing.T) {
	cases := []struct {
		field *Field
		want  string
	}{
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBool}, Offset: 8}}, "s.ReadBool(idx, 8)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt8}, Offset: 9}}, "s.ReadInt8(idx, 9)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint8}, Offset: 10}}, "s.ReadUint8(idx, 10)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt16}, Offset: 12}}, "s.ReadInt16(idx, 12)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint16}, Offset: 14}}, "s.ReadUint16(idx, 14)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}, Offset: 16}}, "s.ReadInt32(idx, 16)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint32}, Offset: 20}}, "s.ReadUint32(idx, 20)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt64}, Offset: 24}}, "s.ReadInt64(idx, 24)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint64}, Offset: 32}}, "s.ReadUint64(idx, 32)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldFloat32}, Offset: 40}}, "s.ReadFloat32(idx, 40)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldFloat64}, Offset: 48}}, "s.ReadFloat64(idx, 48)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString, MaxSize: 32}, Offset: 56, Size: 36}}, "s.ReadString(idx, 56, 36, 32)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes, MaxSize: 64}, Offset: 92, Size: 68}}, "s.ReadBytes(idx, 92, 68, 64)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldType(99)}, Offset: 0}}, "nil, nil // unsupported type"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("ReadCall_%d", tc.field.Type), func(t *testing.T) {
			if got := tc.field.ReadCall(); got != tc.want {
				t.Errorf("ReadCall() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestField_WriteCall(t *testing.T) {
	cases := []struct {
		field *Field
		want  string
	}{
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBool}, Offset: 8}}, "s.WriteBool(idx, 8, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt8}, Offset: 9}}, "s.WriteInt8(idx, 9, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint8}, Offset: 10}}, "s.WriteUint8(idx, 10, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt16}, Offset: 12}}, "s.WriteInt16(idx, 12, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint16}, Offset: 14}}, "s.WriteUint16(idx, 14, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt32}, Offset: 16}}, "s.WriteInt32(idx, 16, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint32}, Offset: 20}}, "s.WriteUint32(idx, 20, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldInt64}, Offset: 24}}, "s.WriteInt64(idx, 24, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldUint64}, Offset: 32}}, "s.WriteUint64(idx, 32, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldFloat32}, Offset: 40}}, "s.WriteFloat32(idx, 40, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldFloat64}, Offset: 48}}, "s.WriteFloat64(idx, 48, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldString, MaxSize: 32}, Offset: 56, Size: 36}}, "s.WriteString(idx, 56, 36, 32, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldBytes, MaxSize: 64}, Offset: 92, Size: 68}}, "s.WriteBytes(idx, 92, 68, 64, val)"},
		{&Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: mmapforge.FieldType(99)}, Offset: 0}}, "nil // unsupported type"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("WriteCall_%d", tc.field.Type), func(t *testing.T) {
			if got := tc.field.WriteCall(); got != tc.want {
				t.Errorf("WriteCall() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestField_WriteCallRec(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{GoName: "Score", Type: mmapforge.FieldUint64}, Offset: 32}}
	want := "s.WriteUint64(idx, 32, rec.Score)"
	if got := f.WriteCallRec(); got != want {
		t.Errorf("WriteCallRec() = %q, want %q", got, want)
	}
}

func TestField_WriteCallRec_String(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{GoName: "Name", Type: mmapforge.FieldString, MaxSize: 32}, Offset: 8, Size: 36}}
	want := "s.WriteString(idx, 8, 36, 32, rec.Name)"
	if got := f.WriteCallRec(); got != want {
		t.Errorf("WriteCallRec() = %q, want %q", got, want)
	}
}

func TestField_WriteCallRec_Default(t *testing.T) {
	f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{GoName: "X", Type: mmapforge.FieldType(99)}}}
	want := "nil // unsupported type"
	if got := f.WriteCallRec(); got != want {
		t.Errorf("WriteCallRec() = %q, want %q", got, want)
	}
}

func TestField_TestValue(t *testing.T) {
	cases := []struct {
		want string
		typ  mmapforge.FieldType
	}{
		{"true", mmapforge.FieldBool},
		{"int8(42)", mmapforge.FieldInt8},
		{"uint8(200)", mmapforge.FieldUint8},
		{"int16(-1234)", mmapforge.FieldInt16},
		{"uint16(54321)", mmapforge.FieldUint16},
		{"int32(-100000)", mmapforge.FieldInt32},
		{"uint32(3000000000)", mmapforge.FieldUint32},
		{"int64(-9000000000)", mmapforge.FieldInt64},
		{"uint64(18000000000000)", mmapforge.FieldUint64},
		{"float32(1.5)", mmapforge.FieldFloat32},
		{"float64(2.5)", mmapforge.FieldFloat64},
		{`"hello"`, mmapforge.FieldString},
		{"[]byte{1, 2, 3}", mmapforge.FieldBytes},
		{"nil", mmapforge.FieldType(99)},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("TestValue_%d", tc.typ), func(t *testing.T) {
			f := &Field{mmapforge.FieldLayout{FieldDef: mmapforge.FieldDef{Type: tc.typ}}}
			if got := f.TestValue(); got != tc.want {
				t.Errorf("TestValue() = %q, want %q", got, tc.want)
			}
		})
	}
}
