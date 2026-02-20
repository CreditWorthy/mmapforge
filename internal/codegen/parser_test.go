package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/CreditWorthy/mmapforge"
)

func writeTempGo(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.go")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseFile_Basic(t *testing.T) {
	src := `package game

// mmapforge:schema version=3
type Player struct {
	ID    uint64
	Score int32
	Name  string ` + "`mmap:\"name,,32\"`" + `
}
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 1 {
		t.Fatalf("got %d schemas, want 1", len(schemas))
	}
	s := schemas[0]
	if s.Name != "Player" {
		t.Errorf("Name = %q, want Player", s.Name)
	}
	if s.Package != "game" {
		t.Errorf("Package = %q, want game", s.Package)
	}
	if s.SchemaVersion != 3 {
		t.Errorf("SchemaVersion = %d, want 3", s.SchemaVersion)
	}
	if len(s.Fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(s.Fields))
	}
	if s.Fields[0].Type != mmapforge.FieldUint64 {
		t.Errorf("field 0 type = %v, want FieldUint64", s.Fields[0].Type)
	}
	if s.Fields[2].Name != "name" {
		t.Errorf("field 2 name = %q, want name", s.Fields[2].Name)
	}
	if s.Fields[2].MaxSize != 32 {
		t.Errorf("field 2 MaxSize = %d, want 32", s.Fields[2].MaxSize)
	}
}

func TestParseFile_InvalidSyntax(t *testing.T) {
	path := writeTempGo(t, `not valid go`)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for invalid Go source")
	}
}

func TestParseFile_NoDirective(t *testing.T) {
	src := `package x
type Foo struct { A int32 }
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 0 {
		t.Fatalf("got %d schemas, want 0", len(schemas))
	}
}

func TestParseFile_NonTypeDecl(t *testing.T) {
	src := `package x
var X = 1
func Foo() {}
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 0 {
		t.Fatalf("got %d schemas, want 0", len(schemas))
	}
}

func TestParseFile_TypeAlias(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type MyInt int32
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 0 {
		t.Fatalf("got %d schemas, want 0 for non-struct type", len(schemas))
	}
}

func TestParseFile_FieldParseError(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type Bad struct {
	X complex128
}
`
	_, err := ParseFile(writeTempGo(t, src))
	if err == nil {
		t.Fatal("expected error for unsupported field type")
	}
}

func TestParseFile_MultipleSchemas(t *testing.T) {
	src := `package x

// mmapforge:schema version=1
type A struct { X int32 }

// mmapforge:schema version=2
type B struct { Y uint64 }
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 2 {
		t.Fatalf("got %d schemas, want 2", len(schemas))
	}
	if schemas[0].SchemaVersion != 1 || schemas[1].SchemaVersion != 2 {
		t.Error("schema versions mismatch")
	}
}

func TestParseFile_FloatingComment(t *testing.T) {
	src := `package x

// mmapforge:schema version=5
type Floated struct {
	V float64
}
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 1 {
		t.Fatalf("got %d schemas, want 1", len(schemas))
	}
	if schemas[0].SchemaVersion != 5 {
		t.Errorf("version = %d, want 5", schemas[0].SchemaVersion)
	}
}

func TestParseFile_EmbeddedFieldSkipped(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type E struct {
	int32
	Name string ` + "`mmap:\"name,,16\"`" + `
}
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas[0].Fields) != 1 {
		t.Fatalf("got %d fields, want 1 (embedded should be skipped)", len(schemas[0].Fields))
	}
}

func TestParseFile_StringWithoutMaxSize(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type S struct { Name string }
`
	_, err := ParseFile(writeTempGo(t, src))
	if err == nil {
		t.Fatal("expected error for string without max_size")
	}
}

func TestParseFile_BytesWithoutMaxSize(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type S struct { Data []byte }
`
	_, err := ParseFile(writeTempGo(t, src))
	if err == nil {
		t.Fatal("expected error for []byte without max_size")
	}
}

func TestParseFile_BadMmapTag(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type S struct { V int32 ` + "`mmap:\"v,,notanumber\"`" + ` }
`
	_, err := ParseFile(writeTempGo(t, src))
	if err == nil {
		t.Fatal("expected error for bad max_size in tag")
	}
}

func TestParseVersionFromDirective(t *testing.T) {
	cases := []struct {
		text   string
		wantV  uint32
		wantOK bool
	}{
		{"// mmapforge:schema version=1", 1, true},
		{"//mmapforge:schema version=42", 42, true},
		{"//  mmapforge:schema  version=100", 100, true},
		{"// not a directive", 0, false},
		{"// mmapforge:schema", 0, false},
		{"// mmapforge:schema version=abc", 0, false},
		{"// mmapforge:schema version=", 0, false},
		{"// something else entirely", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		v, ok := parseVersionFromDirective(tc.text)
		if ok != tc.wantOK || v != tc.wantV {
			t.Errorf("parseVersionFromDirective(%q) = (%d, %v), want (%d, %v)",
				tc.text, v, ok, tc.wantV, tc.wantOK)
		}
	}
}

func TestParseMmapTag(t *testing.T) {
	cases := []struct {
		raw     string
		goName  string
		wantN   string
		wantMS  uint32
		wantErr bool
	}{
		{"", "Foo", "foo", 0, false},
		{"bar", "Foo", "bar", 0, false},
		{",", "Foo", "foo", 0, false},
		{",,32", "Foo", "foo", 32, false},
		{"myname,,64", "Foo", "myname", 64, false},
		{"name,opts,128", "Foo", "name", 128, false},
		{",,notanum", "Foo", "", 0, true},
		{"name,,", "Foo", "name", 0, false},
	}
	for _, tc := range cases {
		name, ms, err := parseMmapTag(tc.raw, tc.goName)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseMmapTag(%q, %q) err=%v, wantErr=%v", tc.raw, tc.goName, err, tc.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if name != tc.wantN || ms != tc.wantMS {
			t.Errorf("parseMmapTag(%q, %q) = (%q, %d), want (%q, %d)",
				tc.raw, tc.goName, name, ms, tc.wantN, tc.wantMS)
		}
	}
}

func TestTagValue(t *testing.T) {
	cases := []struct {
		tag  *ast.BasicLit
		want string
	}{
		{nil, ""},
		{&ast.BasicLit{Value: "`json:\"x\"`"}, ""},
		{&ast.BasicLit{Value: "`mmap:\"hello\"`"}, "hello"},
		{&ast.BasicLit{Value: "`json:\"x\" mmap:\"val,,32\"`"}, "val,,32"},
		{&ast.BasicLit{Value: "`mmap:\"unterminated`"}, ""},
		{&ast.BasicLit{Value: "\"mmap:\\\"x\\\"\""}, ""},
	}
	for i, tc := range cases {
		got := tagValue(tc.tag)
		if got != tc.want {
			t.Errorf("case %d: tagValue() = %q, want %q", i, got, tc.want)
		}
	}
}

func TestGoTypeToFieldType(t *testing.T) {
	valid := []struct {
		goType string
		want   mmapforge.FieldType
	}{
		{"bool", mmapforge.FieldBool},
		{"int8", mmapforge.FieldInt8},
		{"uint8", mmapforge.FieldUint8},
		{"int16", mmapforge.FieldInt16},
		{"uint16", mmapforge.FieldUint16},
		{"int32", mmapforge.FieldInt32},
		{"uint32", mmapforge.FieldUint32},
		{"int64", mmapforge.FieldInt64},
		{"uint64", mmapforge.FieldUint64},
		{"float32", mmapforge.FieldFloat32},
		{"float64", mmapforge.FieldFloat64},
		{"string", mmapforge.FieldString},
		{"[]byte", mmapforge.FieldBytes},
	}
	for _, tc := range valid {
		got, err := goTypeToFieldType(tc.goType)
		if err != nil {
			t.Errorf("goTypeToFieldType(%q) error: %v", tc.goType, err)
		}
		if got != tc.want {
			t.Errorf("goTypeToFieldType(%q) = %v, want %v", tc.goType, got, tc.want)
		}
	}

	_, err := goTypeToFieldType("complex128")
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestTypeString(t *testing.T) {
	fset := token.NewFileSet()
	mustParseExpr := func(src string) ast.Expr {
		t.Helper()
		f, err := parser.ParseFile(fset, "", "package x\nvar _ "+src, 0)
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		gen, ok := f.Decls[0].(*ast.GenDecl)
		if !ok {
			t.Fatal("expected GenDecl")
		}
		vs, ok := gen.Specs[0].(*ast.ValueSpec)
		if !ok {
			t.Fatal("expected ValueSpec")
		}
		return vs.Type
	}

	cases := []struct {
		src  string
		want string
	}{
		{"int32", "int32"},
		{"[]byte", "[]byte"},
		{"[3]int", "[...]int"},
		{"pkg.Type", "pkg.Type"},
	}
	for _, tc := range cases {
		expr := mustParseExpr(tc.src)
		got := typeString(expr)
		if got != tc.want {
			t.Errorf("typeString(%q) = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestTypeString_Default(t *testing.T) {
	expr := &ast.MapType{
		Key:   &ast.Ident{Name: "string"},
		Value: &ast.Ident{Name: "int"},
	}
	if got := typeString(expr); got != "" {
		t.Errorf("typeString(MapType) = %q, want empty", got)
	}
}

func TestFindDirective_DocComment(t *testing.T) {
	src := `package x

// mmapforge:schema version=7
type T struct { X int32 }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	gen, ok := f.Decls[0].(*ast.GenDecl)
	if !ok {
		t.Fatal("expected GenDecl")
	}
	v, ok := findDirective(f, fset, gen, 0)
	if !ok || v != 7 {
		t.Errorf("findDirective() = (%d, %v), want (7, true)", v, ok)
	}
}

func TestFindDirective_NoDirective(t *testing.T) {
	src := `package x

// just a comment
type T struct { X int32 }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	gen, ok := f.Decls[0].(*ast.GenDecl)
	if !ok {
		t.Fatal("expected GenDecl")
	}
	_, ok = findDirective(f, fset, gen, 0)
	if ok {
		t.Error("findDirective() should return false for non-directive comment")
	}
}

func TestFindDirective_NilDoc(t *testing.T) {
	src := `package x
type T struct { X int32 }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	gen, ok := f.Decls[0].(*ast.GenDecl)
	if !ok {
		t.Fatal("expected GenDecl")
	}
	_, ok = findDirective(f, fset, gen, 0)
	if ok {
		t.Error("findDirective() should return false with no comments")
	}
}

func TestFindDirective_FloatingCommentMatch(t *testing.T) {
	src := `package x

var _ = 0 // mmapforge:schema version=9
type T struct { X int32 }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var gen *ast.GenDecl
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if ok && g.Tok == token.TYPE {
			gen = g
			break
		}
	}
	if gen == nil {
		t.Fatal("no type decl found")
	}
	v, ok := findDirective(f, fset, gen, 1)
	if !ok || v != 9 {
		t.Errorf("findDirective() = (%d, %v), want (9, true)", v, ok)
	}
}

func TestExtractSchemas_NonTypeSpec(t *testing.T) {
	fset := token.NewFileSet()
	f := &ast.File{
		Name: &ast.Ident{Name: "x"},
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{{Name: "bogus"}},
					},
				},
			},
		},
	}
	schemas, err := extractSchemas(f, fset)
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) != 0 {
		t.Fatalf("got %d schemas, want 0", len(schemas))
	}
}

func TestParseFile_AllFieldTypes(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type All struct {
	A bool
	B int8
	C uint8
	D int16
	E uint16
	F int32
	G uint32
	H int64
	I uint64
	J float32
	K float64
	L string  ` + "`mmap:\"l,,64\"`" + `
	M []byte  ` + "`mmap:\"m,,128\"`" + `
}
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas[0].Fields) != 13 {
		t.Fatalf("got %d fields, want 13", len(schemas[0].Fields))
	}
	wantTypes := []mmapforge.FieldType{
		mmapforge.FieldBool, mmapforge.FieldInt8, mmapforge.FieldUint8,
		mmapforge.FieldInt16, mmapforge.FieldUint16,
		mmapforge.FieldInt32, mmapforge.FieldUint32,
		mmapforge.FieldInt64, mmapforge.FieldUint64,
		mmapforge.FieldFloat32, mmapforge.FieldFloat64,
		mmapforge.FieldString, mmapforge.FieldBytes,
	}
	for i, wt := range wantTypes {
		if schemas[0].Fields[i].Type != wt {
			t.Errorf("field %d: type = %v, want %v", i, schemas[0].Fields[i].Type, wt)
		}
	}
}

func TestParseFile_DefaultFieldName(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type D struct { MyVal int32 }
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if schemas[0].Fields[0].Name != "myval" {
		t.Errorf("default name = %q, want myval", schemas[0].Fields[0].Name)
	}
}

func TestParseFile_TagEmptyName(t *testing.T) {
	src := `package x
// mmapforge:schema version=1
type D struct { Val int32 ` + "`mmap:\",\"`" + ` }
`
	schemas, err := ParseFile(writeTempGo(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if schemas[0].Fields[0].Name != "val" {
		t.Errorf("empty tag name = %q, want val", schemas[0].Fields[0].Name)
	}
}
