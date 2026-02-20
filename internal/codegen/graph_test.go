package codegen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/CreditWorthy/mmapforge"
)

func TestGenerateFunc_Generate(t *testing.T) {
	called := false
	fn := GenerateFunc(func(_ *Graph) error {
		called = true
		return nil
	})
	err := fn.Generate(&Graph{Config: &Config{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("function should have been called")
	}
}

func TestGenerateFunc_Generate_Error(t *testing.T) {
	fn := GenerateFunc(func(_ *Graph) error {
		return errors.New("fail")
	})
	if err := fn.Generate(&Graph{Config: &Config{}}); err == nil {
		t.Error("expected error")
	}
}

func TestNewGraph_EmptyTarget(t *testing.T) {
	_, err := NewGraph(&Config{Target: ""}, nil)
	if err == nil {
		t.Fatal("expected error for empty target")
	}
}

func TestNewGraph_ComputeLayoutError(t *testing.T) {
	orig := computeLayoutFunc
	defer func() { computeLayoutFunc = orig }()

	computeLayoutFunc = func(_ []mmapforge.FieldDef) (*mmapforge.RecordLayout, error) {
		return nil, errors.New("layout error")
	}

	schemas := []StructSchema{
		{Name: "Foo", Fields: []mmapforge.FieldDef{{Name: "X", Type: mmapforge.FieldUint32}}},
	}
	_, err := NewGraph(&Config{Target: "/tmp/test"}, schemas)
	if err == nil {
		t.Fatal("expected error from ComputeLayout")
	}
}

func TestNewGraph_Success_SchemaPackage(t *testing.T) {
	orig := computeLayoutFunc
	defer func() { computeLayoutFunc = orig }()

	computeLayoutFunc = func(_ []mmapforge.FieldDef) (*mmapforge.RecordLayout, error) {
		return &mmapforge.RecordLayout{
			Fields:     []mmapforge.FieldLayout{{FieldDef: mmapforge.FieldDef{Name: "X", Type: mmapforge.FieldUint32}}},
			RecordSize: 4,
		}, nil
	}

	schemas := []StructSchema{
		{Name: "Foo", Package: "mypkg", Fields: []mmapforge.FieldDef{{Name: "X", Type: mmapforge.FieldUint32}}},
	}
	g, err := NewGraph(&Config{Target: "/tmp/test"}, schemas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, n := range g.Nodes {
		if n != nil && n.Name == "Foo" {
			found = true
			if n.Package != "mypkg" {
				t.Errorf("Package = %q, want %q", n.Package, "mypkg")
			}
		}
	}
	if !found {
		t.Error("Foo node not found")
	}
}

func TestNewGraph_Success_ConfigPackageOverride(t *testing.T) {
	orig := computeLayoutFunc
	defer func() { computeLayoutFunc = orig }()

	computeLayoutFunc = func(_ []mmapforge.FieldDef) (*mmapforge.RecordLayout, error) {
		return &mmapforge.RecordLayout{
			Fields:     []mmapforge.FieldLayout{{FieldDef: mmapforge.FieldDef{Name: "X", Type: mmapforge.FieldUint32}}},
			RecordSize: 4,
		}, nil
	}

	schemas := []StructSchema{
		{Name: "Bar", Package: "original", Fields: []mmapforge.FieldDef{{Name: "X", Type: mmapforge.FieldUint32}}},
	}
	g, err := NewGraph(&Config{Target: "/tmp/test", Package: "override"}, schemas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, n := range g.Nodes {
		if n != nil && n.Name == "Bar" {
			found = true
			if n.Package != "override" {
				t.Errorf("Package = %q, want %q", n.Package, "override")
			}
		}
	}
	if !found {
		t.Error("Bar node not found")
	}
}

func TestNewGraph_MultipleFields(t *testing.T) {
	orig := computeLayoutFunc
	defer func() { computeLayoutFunc = orig }()

	computeLayoutFunc = func(fields []mmapforge.FieldDef) (*mmapforge.RecordLayout, error) {
		layouts := make([]mmapforge.FieldLayout, len(fields))
		for i, f := range fields {
			layouts[i] = mmapforge.FieldLayout{FieldDef: f}
		}
		return &mmapforge.RecordLayout{
			Fields:     layouts,
			RecordSize: 8,
		}, nil
	}

	schemas := []StructSchema{
		{
			Name:    "Multi",
			Package: "pkg",
			Fields: []mmapforge.FieldDef{
				{Name: "A", Type: mmapforge.FieldUint32},
				{Name: "B", Type: mmapforge.FieldUint64},
			},
		},
	}
	g, err := NewGraph(&Config{Target: "/tmp/test"}, schemas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, n := range g.Nodes {
		if n != nil && n.Name == "Multi" {
			if len(n.Fields) != 2 {
				t.Errorf("Fields count = %d, want 2", len(n.Fields))
			}
			return
		}
	}
	t.Error("Multi node not found")
}

func TestGraph_Gen_NoHooks(t *testing.T) {
	called := false
	orig := generateFunc
	defer func() { generateFunc = orig }()

	generateFunc = func(_ *Graph) error {
		called = true
		return nil
	}

	g := &Graph{Config: &Config{}}
	if err := g.Gen(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("generate should have been called")
	}
}

func TestGraph_Gen_WithHooks(t *testing.T) {
	var order []string
	orig := generateFunc
	defer func() { generateFunc = orig }()

	generateFunc = func(_ *Graph) error {
		order = append(order, "generate")
		return nil
	}

	hook1 := func(next Generator) Generator {
		return GenerateFunc(func(g *Graph) error {
			order = append(order, "hook1-before")
			err := next.Generate(g)
			order = append(order, "hook1-after")
			return err
		})
	}
	hook2 := func(next Generator) Generator {
		return GenerateFunc(func(g *Graph) error {
			order = append(order, "hook2-before")
			err := next.Generate(g)
			order = append(order, "hook2-after")
			return err
		})
	}

	g := &Graph{Config: &Config{Hooks: []Hook{hook1, hook2}}}
	if err := g.Gen(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"hook1-before", "hook2-before", "generate", "hook2-after", "hook1-after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], expected[i])
		}
	}
}

func TestGraph_Gen_HookError(t *testing.T) {
	orig := generateFunc
	defer func() { generateFunc = orig }()

	generateFunc = func(_ *Graph) error { return nil }

	hook := func(_ Generator) Generator {
		return GenerateFunc(func(_ *Graph) error {
			return errors.New("hook error")
		})
	}

	g := &Graph{Config: &Config{Hooks: []Hook{hook}}}
	if err := g.Gen(); err == nil {
		t.Error("expected error from hook")
	}
}

func TestGenerate_MkdirAllError(t *testing.T) {
	orig := mkdirAllFunc
	defer func() { mkdirAllFunc = orig }()

	mkdirAllFunc = func(_ string, _ os.FileMode) error {
		return errors.New("mkdir error")
	}

	g := &Graph{Config: &Config{Target: "/tmp/test"}}
	err := generate(g)
	if err == nil {
		t.Fatal("expected error from MkdirAll")
	}
}

func TestGenerate_TypeTemplate_CondSkips(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	GraphTemplates = []GraphTemplate{}
	TypeTemplates = []TypeTemplate{
		{
			Name:   "store",
			Cond:   func(_ *Type) bool { return false },
			Format: func(_ *Type) string { return "skipped.go" },
		},
	}

	g := &Graph{
		Config: &Config{Target: t.TempDir()},
		Nodes:  []*Type{{Config: &Config{}, Name: "Foo", Package: "pkg"}},
	}
	err := generate(g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerate_TypeTemplate_ExecuteError(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	GraphTemplates = []GraphTemplate{}
	TypeTemplates = []TypeTemplate{
		{
			Name:   "nonexistent_template",
			Format: func(_ *Type) string { return "out.go" },
		},
	}

	g := &Graph{
		Config: &Config{Target: t.TempDir()},
		Nodes:  []*Type{{Config: &Config{}, Name: "Foo", Package: "pkg"}},
	}
	err := generate(g)
	if err == nil {
		t.Fatal("expected error from ExecuteTemplate")
	}
}

func TestGenerate_GraphTemplate_SkipTrue(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	TypeTemplates = []TypeTemplate{}
	GraphTemplates = []GraphTemplate{
		{
			Name:   "skipped",
			Skip:   func(_ *Graph) bool { return true },
			Format: "skipped.go",
		},
	}

	g := &Graph{Config: &Config{Target: t.TempDir()}}
	err := generate(g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerate_GraphTemplate_ExecuteError(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	TypeTemplates = []TypeTemplate{}
	GraphTemplates = []GraphTemplate{
		{
			Name:   "nonexistent_graph_template",
			Format: "out.go",
		},
	}

	g := &Graph{Config: &Config{Target: t.TempDir()}}
	err := generate(g)
	if err == nil {
		t.Fatal("expected error from graph ExecuteTemplate")
	}
}

func TestGenerate_TypeTemplate_WriteFormattedError(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	origWrite := writeFileFunc
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
		writeFileFunc = origWrite
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	GraphTemplates = []GraphTemplate{}

	ext := NewTemplate("test")
	if _, err := ext.Parse(`{{ define "testtype" }}package main{{ end }}`); err != nil {
		t.Fatal(err)
	}

	TypeTemplates = []TypeTemplate{
		{
			Name:   "testtype",
			Format: func(_ *Type) string { return "out.go" },
		},
	}

	writeFileFunc = func(_ string, _ []byte, _ os.FileMode) error {
		return errors.New("disk full")
	}

	g := &Graph{
		Config: &Config{
			Target:    t.TempDir(),
			Templates: []*Template{ext},
		},
		Nodes: []*Type{{Config: &Config{}, Name: "Foo", Package: "pkg"}},
	}
	err := generate(g)
	if err == nil {
		t.Fatal("expected writeFormatted error from TypeTemplate loop")
	}
}

func TestGenerate_GraphTemplate_WriteFormattedError(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	origWrite := writeFileFunc
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
		writeFileFunc = origWrite
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	TypeTemplates = []TypeTemplate{}

	ext := NewTemplate("test")
	if _, err := ext.Parse(`{{ define "graphtmpl" }}package main{{ end }}`); err != nil {
		t.Fatal(err)
	}

	GraphTemplates = []GraphTemplate{
		{
			Name:   "graphtmpl",
			Format: "graph_out.go",
		},
	}

	writeFileFunc = func(_ string, _ []byte, _ os.FileMode) error {
		return errors.New("disk full")
	}

	g := &Graph{Config: &Config{
		Target:    t.TempDir(),
		Templates: []*Template{ext},
	}}
	err := generate(g)
	if err == nil {
		t.Fatal("expected writeFormatted error from GraphTemplate loop")
	}
}

func TestGenerate_GraphTemplate_Success(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	origWrite := writeFileFunc
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
		writeFileFunc = origWrite
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	TypeTemplates = []TypeTemplate{}

	ext := NewTemplate("test")
	if _, err := ext.Parse(`{{ define "graphsuccess" }}package main{{ end }}`); err != nil {
		t.Fatal(err)
	}

	GraphTemplates = []GraphTemplate{
		{
			Name:   "graphsuccess",
			Format: "graph_success.go",
		},
	}

	dir := t.TempDir()
	writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
		return os.WriteFile(name, data, perm)
	}

	g := &Graph{Config: &Config{
		Target:    dir,
		Templates: []*Template{ext},
	}}
	err := generate(g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, "graph_success.go")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected graph_success.go to be created")
	}
}

func TestGenerate_ExternalTemplates(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	origWrite := writeFileFunc
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
		writeFileFunc = origWrite
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	GraphTemplates = []GraphTemplate{}

	ext := NewTemplate("ext")
	if _, err := ext.Parse(`{{ define "exttype" }}package main{{ end }}`); err != nil {
		t.Fatal(err)
	}

	TypeTemplates = []TypeTemplate{
		{
			Name:   "exttype",
			Format: func(_ *Type) string { return "ext_out.go" },
		},
	}

	dir := t.TempDir()
	writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
		return os.WriteFile(name, data, perm)
	}

	g := &Graph{
		Config: &Config{
			Target:    dir,
			Templates: []*Template{ext},
		},
		Nodes: []*Type{{Config: &Config{}, Name: "Foo", Package: "pkg"}},
	}
	err := generate(g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, "ext_out.go")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected ext_out.go to be created")
	}
}

func TestGenerate_ExternalTemplates_EmptyTree(t *testing.T) {
	origMkdir := mkdirAllFunc
	origTypes := TypeTemplates
	origGraph := GraphTemplates
	defer func() {
		mkdirAllFunc = origMkdir
		TypeTemplates = origTypes
		GraphTemplates = origGraph
	}()

	mkdirAllFunc = func(_ string, _ os.FileMode) error { return nil }
	TypeTemplates = []TypeTemplate{}
	GraphTemplates = []GraphTemplate{}

	ext := NewTemplate("ext")
	if _, err := ext.Parse(`{{ define "empty" }}{{ end }}`); err != nil {
		t.Fatal(err)
	}

	g := &Graph{
		Config: &Config{
			Target:    t.TempDir(),
			Templates: []*Template{ext},
		},
	}
	err := generate(g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteFormatted_ValidGo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.go")
	src := []byte("package main\n\nfunc main() {}\n")
	if err := writeFormatted(path, src); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if len(data) == 0 {
		t.Error("file should not be empty")
	}
}

func TestWriteFormatted_InvalidGo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.go")
	src := []byte("not valid go code {{{")
	err := writeFormatted(path, src)
	if err == nil {
		t.Fatal("expected error for invalid Go source")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(data) != string(src) {
		t.Error("should have written raw source on format failure")
	}
}

func TestWriteFormatted_FormatError_WriteAlsoFails(t *testing.T) {
	path := "/nonexistent/dir/file.go"
	src := []byte("not valid go {{{")
	err := writeFormatted(path, src)
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := fmt.Sprintf("%v", err)
	if len(errMsg) == 0 {
		t.Error("error message should not be empty")
	}
}

func TestWriteFormatted_WriteError(t *testing.T) {
	origWrite := writeFileFunc
	defer func() { writeFileFunc = origWrite }()

	writeFileFunc = func(_ string, _ []byte, _ os.FileMode) error {
		return errors.New("write error")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "fail.go")
	src := []byte("package main\n\nfunc main() {}\n")
	err := writeFormatted(path, src)
	if err == nil {
		t.Fatal("expected write error")
	}
}
