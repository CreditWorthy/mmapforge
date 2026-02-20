package codegen

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"
	"text/template/parse"
)

//go:embed template/*
var templateDir embed.FS

var defaultFuncMap = template.FuncMap{
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
}

var parseFilesFunc = func(inner *template.Template, filenames ...string) (*template.Template, error) {
	return inner.ParseFiles(filenames...)
}

var addParseTreeFunc = func(inner *template.Template, name string, tree *parse.Tree) (*template.Template, error) {
	return inner.AddParseTree(name, tree)
}

var templates *Template

type Template struct {
	*template.Template
	FuncMap   template.FuncMap
	condition func(*Graph) bool
}

func initTemplates() {
	templates = MustParse(NewTemplate("mmapforge").ParseFS(templateDir, "template/*.tmpl"))
}

func NewTemplate(name string) *Template {
	t := &Template{Template: template.New(name)}
	return t.Funcs(defaultFuncMap)
}

func (t *Template) Funcs(funcMap template.FuncMap) *Template {
	t.Template.Funcs(funcMap)
	if t.FuncMap == nil {
		t.FuncMap = template.FuncMap{}
	}
	for name, f := range funcMap {
		if _, ok := t.FuncMap[name]; !ok {
			t.FuncMap[name] = f
		}
	}
	return t
}

func MustParse(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}

func (t *Template) ParseFS(fsys fs.FS, patterns ...string) (*Template, error) {
	if _, err := t.Template.ParseFS(fsys, patterns...); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Template) SkipIf(cond func(*Graph) bool) *Template {
	t.condition = cond
	return t
}

func (t *Template) Parse(text string) (*Template, error) {
	if _, err := t.Template.Parse(text); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Template) ParseFiles(filenames ...string) (*Template, error) {
	if _, err := parseFilesFunc(t.Template, filenames...); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Template) AddParseTree(name string, tree *parse.Tree) (*Template, error) {
	if _, err := addParseTreeFunc(t.Template, name, tree); err != nil {
		return nil, err
	}
	return t, nil
}
