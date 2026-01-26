package parser

import (
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func TestClassMatcher(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantNil  bool
	}{
		{
			name:     "simple class",
			line:     "class MyClass",
			wantName: "MyClass",
		},
		{
			name:     "class with inheritance",
			line:     "class MyClass < BaseClass",
			wantName: "MyClass",
		},
		{
			name:     "indented class",
			line:     "  class MyClass",
			wantName: "MyClass",
		},
		{
			name:    "not a class",
			line:    "def my_method",
			wantNil: true,
		},
		{
			name:    "lowercase not a class",
			line:    "class myclass",
			wantNil: true,
		},
	}

	matcher := &ClassMatcher{}
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
			if result.Symbols[0].Kind != types.KindClass {
				t.Errorf("expected KindClass, got %v", result.Symbols[0].Kind)
			}
			if result.PushScope != tt.wantName {
				t.Errorf("expected PushScope %q, got %q", tt.wantName, result.PushScope)
			}
		})
	}
}

func TestClassMatcherNestedClass(t *testing.T) {
	matcher := &ClassMatcher{}
	ctx := &ParseContext{
		FilePath: "/test/test.rb",
		LineNum:  1,
	}

	result := matcher.Match("class MyModule::MyClass", ctx)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(result.Symbols))
	}

	sym := result.Symbols[0]
	if sym.Name != "MyClass" {
		t.Errorf("expected name 'MyClass', got %q", sym.Name)
	}
	if len(sym.Scope) != 1 || sym.Scope[0] != "MyModule" {
		t.Errorf("expected scope [MyModule], got %v", sym.Scope)
	}
	if sym.FullName != "MyModule::MyClass" {
		t.Errorf("expected FullName 'MyModule::MyClass', got %q", sym.FullName)
	}
}

func TestClassMatcherWithExistingScope(t *testing.T) {
	matcher := &ClassMatcher{}
	ctx := &ParseContext{
		FilePath:     "/test/test.rb",
		LineNum:      1,
		CurrentScope: []string{"OuterModule"},
	}

	result := matcher.Match("class InnerClass", ctx)
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	sym := result.Symbols[0]
	if sym.FullName != "OuterModule::InnerClass" {
		t.Errorf("expected FullName 'OuterModule::InnerClass', got %q", sym.FullName)
	}
}
