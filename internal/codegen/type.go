package codegen

import (
	"fmt"
	"strings"

	"github.com/CreditWorthy/mmapforge"
)

// Type represents a single mmapforge schema node — the central object
type Type struct {
	*Config

	// Name is the Go struct name from the schema
	Name string

	// Package is the Go package name for the generated file.
	Package string

	// Fields holds the computed field layouts for this type.
	Fields []*Field

	// SchemaVersion is the schema migration version.
	SchemaVersion uint32

	// RecordSize is the total byte size of one record.
	RecordSize uint32
}

// Field wraps mmapforge.FieldLayout and adds template helper methods.
type Field struct {
	mmapforge.FieldLayout
}

// Header returns the file header for generated code.
func (t *Type) Header() string {
	return t.Config.header()
}

// Label returns the snake_case name of the type
func (t *Type) Label() string {
	return strings.ToLower(t.Name)
}

// StoreName returns the generated store struct name
func (t *Type) StoreName() string {
	return t.Name + "Store"
}

// RecordName returns the generated record struct name
func (t *Type) RecordName() string {
	return t.Name + "Record"
}

// LayoutFuncName returns the name of the Layout() func
func (t *Type) LayoutFuncName() string {
	return t.Name + "Layout"
}

// NewStoreFuncName returns the name for CreateStore
func (t *Type) NewStoreFuncName() string {
	return "New" + t.Name + "Store"
}

// OpenStoreFuncName returns the name for OpenStore
func (t *Type) OpenStoreFuncName() string {
	return "Open" + t.Name + "Store"
}

// Receiver returns a short receiver variable name for store methods.
func (t *Type) Receiver() string {
	return "s"
}

// HasStringField reports if any field is a string.
func (t *Type) HasStringField() bool {
	for _, f := range t.Fields {
		if f.IsString() {
			return true
		}
	}
	return false
}

// HasBytesField reports if any field is a bytes field.
func (t *Type) HasBytesField() bool {
	for _, f := range t.Fields {
		if f.IsBytes() {
			return true
		}
	}
	return false
}

// HasVarLenField reports if any field is variable-length (string or bytes).
func (t *Type) HasVarLenField() bool {
	return t.HasStringField() || t.HasBytesField()
}

// GoType returns the Go type string for this field.
func (f *Field) GoType() string {
	switch f.Type {
	case mmapforge.FieldBool:
		return "bool"
	case mmapforge.FieldInt8:
		return "int8"
	case mmapforge.FieldUint8:
		return "uint8"
	case mmapforge.FieldInt16:
		return "int16"
	case mmapforge.FieldUint16:
		return "uint16"
	case mmapforge.FieldInt32:
		return "int32"
	case mmapforge.FieldUint32:
		return "uint32"
	case mmapforge.FieldInt64:
		return "int64"
	case mmapforge.FieldUint64:
		return "uint64"
	case mmapforge.FieldFloat32:
		return "float32"
	case mmapforge.FieldFloat64:
		return "float64"
	case mmapforge.FieldString:
		return "string"
	case mmapforge.FieldBytes:
		return "[]byte"
	default:
		return "unknown"
	}
}

// GetterName returns the name for the getter method
func (f *Field) GetterName() string {
	return "Get" + f.GoName
}

// SetterName returns the name for the setter method
func (f *Field) SetterName() string {
	return "Set" + f.GoName
}

// IsString reports if the field is a string.
func (f *Field) IsString() bool {
	return f.Type == mmapforge.FieldString
}

// IsBytes reports if the field is a []byte.
func (f *Field) IsBytes() bool {
	return f.Type == mmapforge.FieldBytes
}

// IsVarLen reports if the field is variable-length.
func (f *Field) IsVarLen() bool {
	return f.IsString() || f.IsBytes()
}

// IsNumeric reports if the field is a numeric type.
func (f *Field) IsNumeric() bool {
	switch f.Type {
	case mmapforge.FieldInt8, mmapforge.FieldUint8,
		mmapforge.FieldInt16, mmapforge.FieldUint16,
		mmapforge.FieldInt32, mmapforge.FieldUint32,
		mmapforge.FieldInt64, mmapforge.FieldUint64,
		mmapforge.FieldFloat32, mmapforge.FieldFloat64:
		return true
	default:
		return false
	}
}

// IsBool reports if the field is a bool.
func (f *Field) IsBool() bool {
	return f.Type == mmapforge.FieldBool
}

// TypeConstant returns the fmmap.FieldType integer for template use.
func (f *Field) TypeConstant() int {
	return int(f.Type)
}

// ReadCall returns the Store.Read* method call expression for this field.
func (f *Field) ReadCall() string {
	switch f.Type {
	case mmapforge.FieldBool:
		return fmt.Sprintf("s.ReadBool(idx, %d)", f.Offset)
	case mmapforge.FieldInt8:
		return fmt.Sprintf("s.ReadInt8(idx, %d)", f.Offset)
	case mmapforge.FieldUint8:
		return fmt.Sprintf("s.ReadUint8(idx, %d)", f.Offset)
	case mmapforge.FieldInt16:
		return fmt.Sprintf("s.ReadInt16(idx, %d)", f.Offset)
	case mmapforge.FieldUint16:
		return fmt.Sprintf("s.ReadUint16(idx, %d)", f.Offset)
	case mmapforge.FieldInt32:
		return fmt.Sprintf("s.ReadInt32(idx, %d)", f.Offset)
	case mmapforge.FieldUint32:
		return fmt.Sprintf("s.ReadUint32(idx, %d)", f.Offset)
	case mmapforge.FieldInt64:
		return fmt.Sprintf("s.ReadInt64(idx, %d)", f.Offset)
	case mmapforge.FieldUint64:
		return fmt.Sprintf("s.ReadUint64(idx, %d)", f.Offset)
	case mmapforge.FieldFloat32:
		return fmt.Sprintf("s.ReadFloat32(idx, %d)", f.Offset)
	case mmapforge.FieldFloat64:
		return fmt.Sprintf("s.ReadFloat64(idx, %d)", f.Offset)
	case mmapforge.FieldString:
		return fmt.Sprintf("s.ReadString(idx, %d, %d, %d)", f.Offset, f.Size, f.MaxSize)
	case mmapforge.FieldBytes:
		return fmt.Sprintf("s.ReadBytes(idx, %d, %d, %d)", f.Offset, f.Size, f.MaxSize)
	default:
		return "nil, nil // unsupported type"
	}
}

// WriteCall returns the Store.Write* method call using "val" as the value arg.
func (f *Field) WriteCall() string {
	return f.writeCallWith("val")
}

// WriteCallRec returns the Store.Write* method call using "rec.<GoName>" as the value.
func (f *Field) WriteCallRec() string {
	return f.writeCallWith("rec." + f.GoName)
}

func (f *Field) writeCallWith(val string) string {
	switch f.Type {
	case mmapforge.FieldBool:
		return fmt.Sprintf("s.WriteBool(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldInt8:
		return fmt.Sprintf("s.WriteInt8(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldUint8:
		return fmt.Sprintf("s.WriteUint8(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldInt16:
		return fmt.Sprintf("s.WriteInt16(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldUint16:
		return fmt.Sprintf("s.WriteUint16(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldInt32:
		return fmt.Sprintf("s.WriteInt32(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldUint32:
		return fmt.Sprintf("s.WriteUint32(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldInt64:
		return fmt.Sprintf("s.WriteInt64(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldUint64:
		return fmt.Sprintf("s.WriteUint64(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldFloat32:
		return fmt.Sprintf("s.WriteFloat32(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldFloat64:
		return fmt.Sprintf("s.WriteFloat64(idx, %d, %s)", f.Offset, val)
	case mmapforge.FieldString:
		return fmt.Sprintf("s.WriteString(idx, %d, %d, %d, %s)", f.Offset, f.Size, f.MaxSize, val)
	case mmapforge.FieldBytes:
		return fmt.Sprintf("s.WriteBytes(idx, %d, %d, %d, %s)", f.Offset, f.Size, f.MaxSize, val)
	default:
		return "nil // unsupported type"
	}
}
