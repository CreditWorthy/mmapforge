package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/CreditWorthy/mmapforge"
)

type StructSchema struct {
	Name          string
	Package       string
	Fields        []mmapforge.FieldDef
	SchemaVersion uint32
}

func ParseFile(path string) ([]StructSchema, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("mmapforge: parse %s: %w", path, err)
	}

	pkg := f.Name.Name
	var schemas []StructSchema

	for i, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}

		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			version, found := findDirective(f, fset, gen, i)
			if !found {
				continue
			}

			fields, err := parseFields(st)
			if err != nil {
				return nil, fmt.Errorf("mmapforge: struct %s: %w", ts.Name.Name, err)
			}

			schemas = append(schemas, StructSchema{
				Name:          ts.Name.Name,
				Package:       pkg,
				Fields:        fields,
				SchemaVersion: version,
			})
		}
	}

	return schemas, nil
}

func findDirective(f *ast.File, fset *token.FileSet, gen *ast.GenDecl, declIdx int) (uint32, bool) {
	if gen.Doc != nil {
		for _, c := range gen.Doc.List {
			if v, ok := parseVersionFromDirective(c.Text); ok {
				return v, true
			}
		}
	}

	declLine := fset.Position(gen.Pos()).Line
	for _, cg := range f.Comments {
		endLine := fset.Position(cg.End()).Line
		if endLine == declLine-1 || endLine == declLine {
			for _, c := range cg.List {
				if v, ok := parseVersionFromDirective(c.Text); ok {
					return v, true
				}
			}
		}
	}

	_ = declIdx
	return 0, false
}

func parseVersionFromDirective(text string) (uint32, bool) {
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "mmapforge:schema") {
		return 0, false
	}

	parts := strings.Fields(text)
	for _, p := range parts {
		if strings.HasPrefix(p, "version=") {
			vStr := strings.TrimPrefix(p, "version=")
			v, err := strconv.ParseUint(vStr, 10, 32)
			if err != nil {
				return 0, false
			}
			return uint32(v), true
		}
	}
	return 0, false
}

func parseFields(st *ast.StructType) ([]mmapforge.FieldDef, error) {
	var fields []mmapforge.FieldDef
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		goName := field.Names[0].Name
		goType := typeString(field.Type)

		ft, err := goTypeToFieldType(goType)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", goName, err)
		}

		name, maxSize, err := parseMmapTag(tagValue(field.Tag), goName)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", goName, err)
		}

		if (ft == mmapforge.FieldString || ft == mmapforge.FieldBytes) && maxSize == 0 {
			return nil, fmt.Errorf("field %s: max_size required for %s", goName, goType)
		}

		fields = append(fields, mmapforge.FieldDef{
			Name:    name,
			GoName:  goName,
			Type:    ft,
			MaxSize: maxSize,
		})
	}
	return fields, nil
}

// parseMmapTag decodes `mmap:"name,,max_size"`. Returns the mmap field
// name (defaults to lowercase goName) and max_size.
func parseMmapTag(raw string, goName string) (string, uint32, error) {
	if raw == "" {
		return strings.ToLower(goName), 0, nil
	}

	parts := strings.Split(raw, ",")
	name := parts[0]
	if name == "" {
		name = strings.ToLower(goName)
	}

	var maxSize uint32
	if len(parts) >= 3 && parts[2] != "" {
		v, err := strconv.ParseUint(parts[2], 10, 32)
		if err != nil {
			return "", 0, fmt.Errorf("invalid max_size %q: %w", parts[2], err)
		}
		maxSize = uint32(v)
	}
	return name, maxSize, nil
}

// tagValue extracts the value for the "mmap" key from a struct tag literal.
func tagValue(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	raw := tag.Value
	if len(raw) >= 2 && raw[0] == '`' && raw[len(raw)-1] == '`' {
		raw = raw[1 : len(raw)-1]
	}
	const key = `mmap:"`
	idx := strings.Index(raw, key)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(key):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// goTypeToFieldType maps a Go type string to a FieldType.
func goTypeToFieldType(goType string) (mmapforge.FieldType, error) {
	switch goType {
	case "bool":
		return mmapforge.FieldBool, nil
	case "int8":
		return mmapforge.FieldInt8, nil
	case "uint8":
		return mmapforge.FieldUint8, nil
	case "int16":
		return mmapforge.FieldInt16, nil
	case "uint16":
		return mmapforge.FieldUint16, nil
	case "int32":
		return mmapforge.FieldInt32, nil
	case "uint32":
		return mmapforge.FieldUint32, nil
	case "int64":
		return mmapforge.FieldInt64, nil
	case "uint64":
		return mmapforge.FieldUint64, nil
	case "float32":
		return mmapforge.FieldFloat32, nil
	case "float64":
		return mmapforge.FieldFloat64, nil
	case "string":
		return mmapforge.FieldString, nil
	case "[]byte":
		return mmapforge.FieldBytes, nil
	default:
		return 0, fmt.Errorf("unsupported type %q", goType)
	}
}

// typeString converts an AST type expression to its string representation.
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return "[...]" + typeString(t.Elt)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}
