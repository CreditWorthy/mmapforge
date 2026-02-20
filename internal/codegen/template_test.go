package codegen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"text/template"
	"text/template/parse"
)

func TestInitTemplates(t *testing.T) {
	templates = nil
	initTemplates()
	if templates == nil {
		t.Fatal("templates should not be nil after initTemplates")
	}
}

func TestNewTemplate(t *testing.T) {
	tmpl := NewTemplate("test")
	if tmpl == nil {
		t.Fatal("NewTemplate returned nil")
	}
	if tmpl.Template == nil {
		t.Fatal("inner template should not be nil")
	}
	if tmpl.FuncMap == nil {
		t.Fatal("FuncMap should not be nil")
	}
	if _, ok := tmpl.FuncMap["lower"]; !ok {
		t.Error("FuncMap missing 'lower'")
	}
	if _, ok := tmpl.FuncMap["upper"]; !ok {
		t.Error("FuncMap missing 'upper'")
	}
}

func TestTemplate_Funcs_MergesNew(t *testing.T) {
	tmpl := NewTemplate("test")
	custom := template.FuncMap{
		"custom": func() string { return "hi" },
	}
	ret := tmpl.Funcs(custom)
	if ret != tmpl {
		t.Error("Funcs should return the same Template")
	}
	if _, ok := tmpl.FuncMap["custom"]; !ok {
		t.Error("FuncMap missing 'custom'")
	}
}

func TestTemplate_Funcs_DoesNotOverwrite(t *testing.T) {
	tmpl := NewTemplate("test")
	originalLower := tmpl.FuncMap["lower"]
	tmpl.Funcs(template.FuncMap{
		"lower": func(_ string) string { return "override" },
	})
	if tmpl.FuncMap["lower"] == nil {
		t.Fatal("lower should still exist")
	}
	_ = originalLower
}

func TestTemplate_Funcs_NilFuncMap(t *testing.T) {
	tmpl := &Template{Template: template.New("bare")}
	tmpl.FuncMap = nil
	tmpl.Funcs(template.FuncMap{
		"foo": func() string { return "bar" },
	})
	if tmpl.FuncMap == nil {
		t.Fatal("FuncMap should be initialized")
	}
	if _, ok := tmpl.FuncMap["foo"]; !ok {
		t.Error("FuncMap missing 'foo'")
	}
}

func TestMustParse_Success(t *testing.T) {
	tmpl := NewTemplate("test")
	parsed, err := tmpl.Parse("{{ . }}")
	if err != nil {
		t.Fatal(err)
	}
	got := MustParse(parsed, nil)
	if got != parsed {
		t.Error("MustParse should return the template on success")
	}
}

func TestMustParse_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse should panic on error")
		}
	}()
	MustParse(nil, errTest)
}

var errTest = func() error {
	return &testError{}
}()

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestTemplate_ParseFS_Success(t *testing.T) {
	fsys := fstest.MapFS{
		"test.tmpl": &fstest.MapFile{Data: []byte(`{{ define "test" }}hello{{ end }}`)},
	}
	tmpl := NewTemplate("test")
	got, err := tmpl.ParseFS(fsys, "*.tmpl")
	if err != nil {
		t.Fatalf("ParseFS error: %v", err)
	}
	if got != tmpl {
		t.Error("ParseFS should return the same Template")
	}
}

func TestTemplate_ParseFS_Error(t *testing.T) {
	fsys := fstest.MapFS{}
	tmpl := NewTemplate("test")
	_, err := tmpl.ParseFS(fsys, "nonexistent/*.tmpl")
	if err == nil {
		t.Error("ParseFS should return error for missing pattern")
	}
}

func TestTemplate_SkipIf(t *testing.T) {
	tmpl := NewTemplate("test")
	if tmpl.condition != nil {
		t.Error("condition should be nil initially")
	}
	cond := func(_ *Graph) bool { return true }
	ret := tmpl.SkipIf(cond)
	if ret != tmpl {
		t.Error("SkipIf should return the same Template")
	}
	if tmpl.condition == nil {
		t.Error("condition should be set")
	}
	if !tmpl.condition(&Graph{Config: &Config{}}) {
		t.Error("condition should return true")
	}
}

func TestTemplate_Parse_Success(t *testing.T) {
	tmpl := NewTemplate("test")
	got, err := tmpl.Parse("hello {{ lower . }}")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got != tmpl {
		t.Error("Parse should return the same Template")
	}
}

func TestTemplate_Parse_Error(t *testing.T) {
	tmpl := NewTemplate("test")
	_, err := tmpl.Parse("{{ .Broken")
	if err == nil {
		t.Error("Parse should return error for invalid template")
	}
}

func TestTemplate_ParseFiles_Error(t *testing.T) {
	tmpl := NewTemplate("test")
	_, err := tmpl.ParseFiles("/nonexistent/file.tmpl")
	if err == nil {
		t.Error("ParseFiles should return error for missing file")
	}
}

func TestTemplate_AddParseTree_Success(t *testing.T) {
	tmpl := NewTemplate("test")
	if _, err := tmpl.Parse("base"); err != nil {
		t.Fatal(err)
	}

	tree := &parse.Tree{
		Name: "sub",
		Root: &parse.ListNode{
			NodeType: parse.NodeList,
		},
	}
	got, err := tmpl.AddParseTree("sub", tree)
	if err != nil {
		t.Fatalf("AddParseTree error: %v", err)
	}
	if got != tmpl {
		t.Error("AddParseTree should return the same Template")
	}
}

func TestDefaultFuncMap_Lower(t *testing.T) {
	fn, ok := defaultFuncMap["lower"]
	if !ok {
		t.Fatal("defaultFuncMap missing 'lower'")
	}
	lower, ok := fn.(func(string) string)
	if !ok {
		t.Fatal("lower is not func(string) string")
	}
	if got := lower("HELLO"); got != "hello" {
		t.Errorf("lower(HELLO) = %q, want %q", got, "hello")
	}
}

func TestDefaultFuncMap_Upper(t *testing.T) {
	fn, ok := defaultFuncMap["upper"]
	if !ok {
		t.Fatal("defaultFuncMap missing 'upper'")
	}
	upper, ok := fn.(func(string) string)
	if !ok {
		t.Fatal("upper is not func(string) string")
	}
	if got := upper("hello"); got != "HELLO" {
		t.Errorf("upper(hello) = %q, want %q", got, "HELLO")
	}
}

func TestTemplate_ParseFiles_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tmpl")
	if err := os.WriteFile(path, []byte(`{{ define "hello" }}world{{ end }}`), 0600); err != nil {
		t.Fatal(err)
	}

	tmpl := NewTemplate("test")
	got, err := tmpl.ParseFiles(path)
	if err != nil {
		t.Fatalf("ParseFiles error: %v", err)
	}
	if got != tmpl {
		t.Error("ParseFiles should return the same Template")
	}
}

func TestTemplate_AddParseTree_Error(t *testing.T) {
	tmpl := NewTemplate("test")
	orig := addParseTreeFunc
	defer func() { addParseTreeFunc = orig }()

	addParseTreeFunc = func(_ *template.Template, _ string, _ *parse.Tree) (*template.Template, error) {
		return nil, errors.New("mock add parse tree error")
	}

	tree := &parse.Tree{
		Name: "sub",
		Root: &parse.ListNode{NodeType: parse.NodeList},
	}
	got, err := tmpl.AddParseTree("sub", tree)
	if err == nil {
		t.Fatal("AddParseTree should return error")
	}
	if got != nil {
		t.Error("AddParseTree should return nil on error")
	}
}
