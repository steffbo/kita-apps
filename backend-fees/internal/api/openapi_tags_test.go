package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// swag runs with --requiredByDefault (scripts/generate-api.sh), so every JSON
// field ends up "required" in the OpenAPI spec unless tagged binding:"optional".
// Fields the API may leave out (omitempty) or send as null (pointers) must
// carry that tag, otherwise the generated frontend types claim they are
// always present.
func TestOptionalJSONFieldsAreTaggedForOpenAPI(t *testing.T) {
	root := filepath.Join("..")
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				if field.Tag == nil || len(field.Names) == 0 {
					continue
				}
				raw, _ := strconv.Unquote(field.Tag.Value)
				tag := reflect.StructTag(raw)
				jsonTag, ok := tag.Lookup("json")
				if !ok || jsonTag == "-" {
					continue
				}
				_, isPointer := field.Type.(*ast.StarExpr)
				if !strings.Contains(jsonTag, ",omitempty") && !isPointer {
					continue
				}
				if _, tagged := tag.Lookup("binding"); !tagged {
					t.Errorf("%s: field %s (json:%q) needs binding:\"optional\" for the OpenAPI spec",
						fset.Position(field.Pos()), field.Names[0].Name, jsonTag)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
