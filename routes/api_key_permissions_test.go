package routes

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/go-playground/validator/v10"
)

// Every permission required by a route must be grantable through both key DTOs.
// This catches drift such as requiring "create" while keys only accept "write".
func TestRoutePermissionsCanBeGranted(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "short_routes.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	validate := validator.New()
	checked := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "CheckPermissionAPIKey" {
			return true
		}
		permissions, ok := call.Args[1].(*ast.CompositeLit)
		if !ok {
			t.Fatal("expected explicit route permissions")
		}
		for _, element := range permissions.Elts {
			literal, ok := element.(*ast.BasicLit)
			if !ok {
				t.Fatal("expected literal permission")
			}
			permission, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			for _, request := range []any{dto.CreateAPIKeyRequest{}, dto.UpdateAPIKeyRequest{}} {
				field, _ := reflect.TypeOf(request).FieldByName("Permissions")
				if err := validate.Var([]string{permission}, field.Tag.Get("binding")); err != nil {
					t.Errorf("route permission %q cannot be granted by %T: %v", permission, request, err)
				}
			}
			checked++
		}
		return true
	})
	if checked == 0 {
		t.Fatal("no API-key route permissions checked")
	}
}
