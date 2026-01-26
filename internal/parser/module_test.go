package parser

import (
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func TestModuleMatcher(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantNil  bool
	}{
		{
			name:     "simple module",
			line:     "module MyModule",
			wantName: "MyModule",
		},
		{
			name:     "indented module",
			line:     "  module MyModule",
			wantName: "MyModule",
		},
		{
			name:    "not a module",
			line:    "class MyClass",
			wantNil: true,
		},
		{
			name:    "lowercase not a module",
			line:    "module mymodule",
			wantNil: true,
		},
	}

	matcher := &ModuleMatcher{}
	ctx := &ParseContext{
		FilePath: "/test/test.rb",
		LineNum:  1,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.line, ctx)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if len(result.Symbols) != 1 {
				t.Fatalf("expected 1 symbol, got %d", len(result.Symbols))
			}
			if result.Symbols[0].Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Symbols[0].Name)
			}
			if result.Symbols[0].Kind != types.KindModule {
				t.Errorf("expected KindModule, got %v", result.Symbols[0].Kind)
			}
			if result.PushScope != tt.wantName {
				t.Errorf("expected PushScope %q, got %q", tt.wantName, result.PushScope)
			}
		})
	}
}

func TestModuleMatcherNestedModule(t *testing.T) {
	matcher := &ModuleMatcher{}
	ctx := &ParseContext{
		FilePath: "/test/test.rb",
		LineNum:  1,
	}

	result := matcher.Match("module MyParent::MyModule", ctx)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(result.Symbols))
	}

	sym := result.Symbols[0]
	if sym.Name != "MyModule" {
		t.Errorf("expected name 'MyModule', got %q", sym.Name)
	}
	if len(sym.Scope) != 1 || sym.Scope[0] != "MyParent" {
		t.Errorf("expected scope [MyParent], got %v", sym.Scope)
	}
	if sym.FullName != "MyParent::MyModule" {
		t.Errorf("expected FullName 'MyParent::MyModule', got %q", sym.FullName)
	}
}

func TestModuleMatcherWithExistingScope(t *testing.T) {
	matcher := &ModuleMatcher{}
	ctx := &ParseContext{
		FilePath:     "/test/test.rb",
		LineNum:      1,
		CurrentScope: []string{"OuterModule"},
	}

	result := matcher.Match("module InnerModule", ctx)
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	sym := result.Symbols[0]
	if sym.FullName != "OuterModule::InnerModule" {
		t.Errorf("expected FullName 'OuterModule::InnerModule', got %q", sym.FullName)
	}
}
