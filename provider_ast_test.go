// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvidersDoNotSerializePublicRequestsDirectly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Clean(entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			name := calledFunctionName(call.Fun)
			if name != "Marshal" && name != "Encode" && name != "newJSONRequest" && name != "mergeExtraBody" && name != "doJSON" {
				return true
			}
			for _, argument := range call.Args {
				identifier, ok := argument.(*ast.Ident)
				if ok && (identifier.Name == "request" || identifier.Name == "req") {
					t.Errorf("%s directly passes public request to %s", path, name)
				}
			}
			return true
		})
	}
}

func TestPublicErrorContract(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "errors.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{}{
		"ErrInvalidAPIKeyConfig":     {},
		"ErrCredentialSelection":     {},
		"ErrInvalidRequest":          {},
		"ErrUnsupportedProvider":     {},
		"UnsupportedCapabilityError": {},
		"APIError":                   {},
	}
	got := make(map[string]struct{})
	for _, declaration := range file.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok || (generic.Tok != token.VAR && generic.Tok != token.TYPE) {
			continue
		}
		for _, specification := range generic.Specs {
			switch value := specification.(type) {
			case *ast.ValueSpec:
				for _, name := range value.Names {
					if ast.IsExported(name.Name) {
						got[name.Name] = struct{}{}
					}
				}
			case *ast.TypeSpec:
				if ast.IsExported(value.Name.Name) {
					got[value.Name.Name] = struct{}{}
				}
			}
		}
	}
	for name := range want {
		if _, exists := got[name]; !exists {
			t.Errorf("public error contract is missing %s", name)
		}
	}
	for name := range got {
		if _, exists := want[name]; !exists {
			t.Errorf("errors.go unexpectedly exports %s", name)
		}
	}
}

func calledFunctionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}
