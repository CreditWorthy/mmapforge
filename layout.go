package mmapforge

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
)

const SeqFieldSize = 8

// FieldType enumerates the supported binary field types.
type FieldType int

const (
	FieldBool FieldType = iota
	FieldInt8
	FieldUint8
	FieldInt16
	FieldUint16
	FieldInt32
	FieldUint32
	FieldInt64
	FieldUint64
	FieldFloat32
	FieldFloat64
	FieldString
	FieldBytes
)

// FieldDef is the input to the layout engine: one per struct field.
type FieldDef struct {
	Name    string
	GoName  string
	Type    FieldType
	MaxSize uint32
}

// FieldLayout is the output: a field with its computed offset and size.
type FieldLayout struct {
	FieldDef
	Offset uint32
	Size   uint32
	Align  uint32
}

// RecordLayout is the complete layout for one struct.
type RecordLayout struct {
	Fields     []FieldLayout
	RecordSize uint32
}

// ComputeLayout takes field definitions in declaration order and returns
// the byte layout with proper alignment. Returns an error if any field
// definition is invalid.
//
// The first 8 bytes of every record are reserved for the seqlock
// sequence counter. User fields start at offset 8.
func ComputeLayout(fields []FieldDef) (*RecordLayout, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("mmapforge: layout: no fields")
	}

	type fieldMeta struct {
		def   FieldDef
		size  uint32
		align uint32
	}

	metas := make([]fieldMeta, len(fields))
	seen := make(map[string]bool, len(fields))

	for i, f := range fields {
		if seen[f.Name] {
			return nil, fmt.Errorf("mmapforge: layout: duplicate field name %q", f.Name)
		}
		seen[f.Name] = true
		size, align, err := fieldSizeAlign(f)
		if err != nil {
			return nil, fmt.Errorf("mmapforge: layout: field %q: %w", f.Name, err)
		}
		metas[i] = fieldMeta{def: f, size: size, align: align}
	}

	layouts := make([]FieldLayout, len(metas))
	var offset uint32 = SeqFieldSize

	for i, m := range metas {
		if rem := offset % m.align; rem != 0 {
			offset += m.align - rem
		}

		layouts[i] = FieldLayout{
			FieldDef: m.def,
			Offset:   offset,
			Size:     m.size,
			Align:    m.align,
		}
		offset += m.size
	}

	recordSize := offset
	if rem := offset % 8; rem != 0 {
		recordSize += 8 - rem
	}

	return &RecordLayout{
		Fields:     layouts,
		RecordSize: recordSize,
	}, nil
}

// fieldSizeAlign returns (size, alignment) for a field.
func fieldSizeAlign(f FieldDef) (size, align uint32, err error) {
	switch f.Type {
	case FieldBool, FieldInt8, FieldUint8:
		return 1, 1, nil
	case FieldInt16, FieldUint16:
		return 2, 2, nil
	case FieldInt32, FieldUint32, FieldFloat32:
		return 4, 4, nil
	case FieldInt64, FieldUint64, FieldFloat64:
		return 8, 8, nil
	case FieldString, FieldBytes:
		if f.MaxSize == 0 {
			return 0, 0, fmt.Errorf("max_size required for %v", f.Type)
		}
		if f.MaxSize > math.MaxUint32-4 {
			return 0, 0, fmt.Errorf("max_size %d overflows uint32", f.MaxSize)
		}
		return 4 + f.MaxSize, 4, nil
	default:
		return 0, 0, fmt.Errorf("unknown field type %d", f.Type)
	}
}

// String returns the canonical name for a field type.
func (t FieldType) String() string {
	switch t {
	case FieldBool:
		return "bool"
	case FieldInt8:
		return "int8"
	case FieldUint8:
		return "uint8"
	case FieldInt16:
		return "int16"
	case FieldUint16:
		return "uint16"
	case FieldInt32:
		return "int32"
	case FieldUint32:
		return "uint32"
	case FieldInt64:
		return "int64"
	case FieldUint64:
		return "uint64"
	case FieldFloat32:
		return "float32"
	case FieldFloat64:
		return "float64"
	case FieldString:
		return "string"
	case FieldBytes:
		return "bytes"
	default:
		return "unknown"
	}
}

// FieldDescriptor is the canonical representation of a field for schema hashing.
type FieldDescriptor struct {
	Name string
	Type string
	Size uint32
}

// Descriptors converts the layout to FieldDescriptors for schema hashing
func (r *RecordLayout) Descriptors() []FieldDescriptor {
	descs := make([]FieldDescriptor, len(r.Fields))
	for i, f := range r.Fields {
		descs[i] = FieldDescriptor{
			Name: f.Name,
			Type: f.Type.String(),
			Size: f.Size,
		}
	}
	return descs
}

// SchemaHash computes the SHA-256 of a canonical field descriptor string.
// Fields are sorted by name so the hash is layout-order-independent
func SchemaHash(fields []FieldDescriptor) [32]byte {
	sorted := make([]FieldDescriptor, len(fields))
	copy(sorted, fields)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	parts := make([]string, len(sorted))
	for i, f := range sorted {
		parts[i] = fmt.Sprintf("%s:%s:%d", f.Name, f.Type, f.Size)
	}
	return sha256.Sum256([]byte(strings.Join(parts, ",")))
}
