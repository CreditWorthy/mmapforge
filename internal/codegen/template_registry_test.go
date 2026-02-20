package codegen

import "testing"

func TestTypeTemplate_Format(t *testing.T) {
	if len(TypeTemplates) == 0 {
		t.Fatal("TypeTemplates should not be empty")
	}

	tmpl := TypeTemplates[0]

	if tmpl.Name != "store" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "store")
	}

	typ := &Type{Name: "Account"}
	got := tmpl.Format(typ)
	if got != "account_store.go" {
		t.Errorf("Format() = %q, want %q", got, "account_store.go")
	}
}

func TestTypeTemplate_Cond_Nil(t *testing.T) {
	tmpl := TypeTemplates[0]
	if tmpl.Cond != nil {
		t.Error("default store template Cond should be nil")
	}
}

func TestTypeTemplate_Cond_True(t *testing.T) {
	tmpl := TypeTemplate{
		Cond:   func(_ *Type) bool { return true },
		Format: func(_ *Type) string { return "test.go" },
		Name:   "test",
	}
	if !tmpl.Cond(&Type{}) {
		t.Error("Cond should return true")
	}
	if tmpl.Name != "test" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "test")
	}
	if got := tmpl.Format(&Type{}); got != "test.go" {
		t.Errorf("Format = %q, want %q", got, "test.go")
	}
}

func TestTypeTemplate_Cond_False(t *testing.T) {
	tmpl := TypeTemplate{
		Cond:   func(_ *Type) bool { return false },
		Format: func(_ *Type) string { return "test.go" },
		Name:   "test",
	}
	if tmpl.Cond(&Type{}) {
		t.Error("Cond should return false")
	}
	if tmpl.Name != "test" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "test")
	}
	if got := tmpl.Format(&Type{}); got != "test.go" {
		t.Errorf("Format = %q, want %q", got, "test.go")
	}
}

func TestGraphTemplate_Skip_Nil(t *testing.T) {
	tmpl := GraphTemplate{
		Name:   "graph",
		Format: "graph.go",
	}
	if tmpl.Skip != nil {
		t.Error("Skip should be nil")
	}
	if tmpl.Name != "graph" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "graph")
	}
	if tmpl.Format != "graph.go" {
		t.Errorf("Format = %q, want %q", tmpl.Format, "graph.go")
	}
}

func TestGraphTemplate_Skip_True(t *testing.T) {
	tmpl := GraphTemplate{
		Name:   "graph",
		Skip:   func(_ *Graph) bool { return true },
		Format: "graph.go",
	}
	if !tmpl.Skip(&Graph{Config: &Config{}}) {
		t.Error("Skip should return true")
	}
	if tmpl.Name != "graph" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "graph")
	}
	if tmpl.Format != "graph.go" {
		t.Errorf("Format = %q, want %q", tmpl.Format, "graph.go")
	}
}

func TestGraphTemplate_Skip_False(t *testing.T) {
	tmpl := GraphTemplate{
		Name:   "graph",
		Skip:   func(_ *Graph) bool { return false },
		Format: "graph.go",
	}
	if tmpl.Skip(&Graph{Config: &Config{}}) {
		t.Error("Skip should return false")
	}
	if tmpl.Name != "graph" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "graph")
	}
	if tmpl.Format != "graph.go" {
		t.Errorf("Format = %q, want %q", tmpl.Format, "graph.go")
	}
}

func TestGraphTemplates_Empty(t *testing.T) {
	if len(GraphTemplates) != 0 {
		t.Errorf("GraphTemplates should be empty, got %d", len(GraphTemplates))
	}
}
