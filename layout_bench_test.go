package mmapforge

import (
	"testing"
)

func BenchmarkComputeLayout_Small(b *testing.B) {
	fields := []FieldDef{
		{Name: "id", Type: FieldUint64},
		{Name: "value", Type: FieldFloat64},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, layoutErr := ComputeLayout(fields); layoutErr != nil {
			b.Fatal(layoutErr)
		}
	}
}

func BenchmarkComputeLayout_Medium(b *testing.B) {
	fields := []FieldDef{
		{Name: "id", Type: FieldUint64},
		{Name: "value", Type: FieldFloat64},
		{Name: "score", Type: FieldInt32},
		{Name: "flags", Type: FieldUint8},
		{Name: "name", Type: FieldString, MaxSize: 32},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, layoutErr := ComputeLayout(fields); layoutErr != nil {
			b.Fatal(layoutErr)
		}
	}
}

func BenchmarkComputeLayout_Large(b *testing.B) {
	fields := []FieldDef{
		{Name: "id", Type: FieldUint64},
		{Name: "ts", Type: FieldInt64},
		{Name: "price", Type: FieldFloat64},
		{Name: "volume", Type: FieldFloat64},
		{Name: "qty", Type: FieldInt32},
		{Name: "side", Type: FieldUint8},
		{Name: "active", Type: FieldBool},
		{Name: "tag", Type: FieldUint16},
		{Name: "name", Type: FieldString, MaxSize: 64},
		{Name: "data", Type: FieldBytes, MaxSize: 128},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, layoutErr := ComputeLayout(fields); layoutErr != nil {
			b.Fatal(layoutErr)
		}
	}
}

func BenchmarkSchemaHash(b *testing.B) {
	layout, layoutErr := ComputeLayout([]FieldDef{
		{Name: "id", Type: FieldUint64},
		{Name: "value", Type: FieldFloat64},
		{Name: "score", Type: FieldInt32},
		{Name: "flags", Type: FieldUint8},
		{Name: "name", Type: FieldString, MaxSize: 32},
	})
	if layoutErr != nil {
		b.Fatal(layoutErr)
	}
	descs := layout.Descriptors()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SchemaHash(descs)
	}
}
