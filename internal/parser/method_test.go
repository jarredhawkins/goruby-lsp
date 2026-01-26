package parser

import (
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func TestMethodMatcher(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantKind types.SymbolKind
		wantNil  bool
	}{
		{
			name:     "simple method",
			line:     "def my_method",
			wantName: "my_method",
			wantKind: types.KindMethod,
		},
		{
			name:     "method with args",
			line:     "def my_method(arg1, arg2)",
			wantName: "my_method",
			wantKind: types.KindMethod,
		},
		{
			name:     "indented method",
			line:     "  def my_method",
			wantName: "my_method",
			wantKind: types.KindMethod,
		},
		{
			name:     "singleton method",
			line:     "def self.my_method",
			wantName: "my_method",
			wantKind: types.KindSingletonMethod,
		},
		{
			name:     "predicate method",
			line:     "def valid?",
			wantName: "valid?",
			wantKind: types.KindMethod,
		},
		{
			name:     "bang method",
			line:     "def save!",
			wantName: "save!",
			wantKind: types.KindMethod,
		},
		{
			name:     "setter method",
			line:     "def name=",
			wantName: "name=",
			wantKind: types.KindMethod,
		},
		{
			name:     "singleton predicate method",
			line:     "def self.valid?",
			wantName: "valid?",
			wantKind: types.KindSingletonMethod,
		},
		{
			name:    "not a method",
			line:    "class MyClass",
			wantNil: true,
		},
	}

	matcher := &MethodMatcher{}
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
			if result.Symbols[0].Kind != tt.wantKind {
				t.Errorf("expected kind %v, got %v", tt.wantKind, result.Symbols[0].Kind)
			}
			if result.EnterMethod == nil {
				t.Error("expected EnterMethod to be set")
			}
		})
	}
}

func TestMethodMatcherWithScope(t *testing.T) {
	matcher := &MethodMatcher{}
	ctx := &ParseContext{
		FilePath:     "/test/test.rb",
		LineNum:      5,
		CurrentScope: []string{"MyModule", "MyClass"},
	}

	result := matcher.Match("def my_method", ctx)
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	sym := result.Symbols[0]
	if sym.FullName != "MyModule::MyClass#my_method" {
		t.Errorf("expected FullName 'MyModule::MyClass#my_method', got %q", sym.FullName)
	}

	if result.EnterMethod.FullName != "MyModule::MyClass#my_method" {
		t.Errorf("expected EnterMethod.FullName 'MyModule::MyClass#my_method', got %q", result.EnterMethod.FullName)
	}
	if result.EnterMethod.StartLine != 5 {
		t.Errorf("expected EnterMethod.StartLine 5, got %d", result.EnterMethod.StartLine)
	}
}
