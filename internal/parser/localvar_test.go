package parser

import (
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func TestLocalVariableParsing(t *testing.T) {
	content := `class MyClass
  def my_method
    x = 1
    y, z = 2, 3
    puts x
    puts y
    puts z
  end

  def another_method
    a = "hello"
    b = "world"
  end
end`

	registry := NewRegistry()
	RegisterDefaults(registry)

	scanner := NewScanner(registry)
	symbols := scanner.Parse("/test/test.rb", []byte(content))

	// Count symbol types
	var classes, methods, localVars int
	for _, sym := range symbols {
		switch sym.Kind {
		case types.KindClass:
			classes++
		case types.KindMethod:
			methods++
		case types.KindLocalVariable:
			localVars++
		}
	}

	t.Logf("Found %d symbols: %d classes, %d methods, %d local variables",
		len(symbols), classes, methods, localVars)

	// Print all symbols for debugging
	for _, sym := range symbols {
		t.Logf("  %s %s at line %d (EndLine: %d) [method: %s]",
			sym.Kind, sym.FullName, sym.Line, sym.EndLine, sym.MethodFullName)
	}

	// Verify counts
	if classes != 1 {
		t.Errorf("Expected 1 class, got %d", classes)
	}
	if methods != 2 {
		t.Errorf("Expected 2 methods, got %d", methods)
	}
	if localVars != 5 {
		t.Errorf("Expected 5 local variables (x, y, z, a, b), got %d", localVars)
	}

	// Verify method EndLine is set
	for _, sym := range symbols {
		if sym.Kind == types.KindMethod {
			if sym.EndLine == 0 {
				t.Errorf("Method %s has no EndLine set", sym.Name)
			}
			t.Logf("Method %s: lines %d-%d", sym.Name, sym.Line, sym.EndLine)
		}
	}

	// Verify local variables are in correct methods
	for _, sym := range symbols {
		if sym.Kind == types.KindLocalVariable {
			if sym.MethodFullName == "" {
				t.Errorf("Local variable %s has no MethodFullName set", sym.Name)
			}
			t.Logf("Local var %s is in method %s", sym.Name, sym.MethodFullName)
		}
	}
}

func TestLocalVariableNotOutsideMethod(t *testing.T) {
	content := `x = 1

class MyClass
  X = 100

  def my_method
    y = 2
  end
end

z = 3`

	registry := NewRegistry()
	RegisterDefaults(registry)

	scanner := NewScanner(registry)
	symbols := scanner.Parse("/test/test.rb", []byte(content))

	// Count local variables
	var localVars int
	for _, sym := range symbols {
		if sym.Kind == types.KindLocalVariable {
			localVars++
			t.Logf("Found local var: %s at line %d in method %s",
				sym.Name, sym.Line, sym.MethodFullName)
		}
	}

	// Should only find 'y' inside the method, not 'x' or 'z' outside
	if localVars != 1 {
		t.Errorf("Expected 1 local variable (y inside method), got %d", localVars)
	}
}
