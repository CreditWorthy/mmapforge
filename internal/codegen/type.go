package codegen

import "github.com/CreditWorthy/mmapforge"

type Type struct {
	*Config

	Name          string
	Package       string
	SchemaVersion uint32
	Fields        []*Field
	RecordSize    uint32
}

type Field struct {
	mmapforge.FieldLayout
}
