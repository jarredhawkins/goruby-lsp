package parser

import (
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func TestConstantMatcher(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantNil  bool
	}{
		{
			name:     "screaming snake case",
			line:     "MY_CONSTANT = 42",
			wantName: "MY_CONSTANT",
		},
		{
			name:     "simple constant",
			line:     "VERSION = '1.0.0'",
			wantName: "VERSION",
		},
		{
			name:     "indented constant",
			line:     "  MY_CONSTANT = 42",
			wantName: "MY_CONSTANT",
		},
		{
			name:     "constant with numbers",
			line:     "API_V2 = true",
			wantName: "API_V2",
		},
		{
			name:    "lowercase not a constant",
			line:    "my_var = 42",
			wantNil: true,
		},
		{
			name:    "mixed case not matched by this pattern",
			line:    "MyClass = Class.new",
			wantNil: true,
		},
		{
			name:    "not an assignment (==)",
			line:    "MY_CONSTANT == 42",
			wantNil: true,
		},
		{
			name:    "not an assignment (===)",
			line:    "MY_CONSTANT === 42",
			wantNil: true,
		},
	}

	matcher := &ConstantMatcher{}
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
			if result.Symbols[0].Kind != types.KindConstant {
				t.Errorf("expected KindConstant, got %v", result.Symbols[0].Kind)
			}
		})
	}
}

func TestConstantMatcherWithScope(t *testing.T) {
	matcher := &ConstantMatcher{}
	ctx := &ParseContext{
		FilePath:     "/test/test.rb",
		LineNum:      1,
		CurrentScope: []string{"MyModule", "MyClass"},
	}

	result := matcher.Match("MY_CONSTANT = 42", ctx)
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	sym := result.Symbols[0]
	if sym.FullName != "MyModule::MyClass::MY_CONSTANT" {
		t.Errorf("expected FullName 'MyModule::MyClass::MY_CONSTANT', got %q", sym.FullName)
	}
}
