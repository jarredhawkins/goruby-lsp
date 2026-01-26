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

func TestLocalVariableWithDoBlocks(t *testing.T) {
	// This tests the fix for do...end block nesting
	// The method EndLine should be set correctly even with nested blocks
	content := `class Worker
  def perform
    items = []
    items.each do |item|
      process(item)
    end
    result = Hash.new(0)
    items.each_with_index do |item, idx|
      result[idx] = item
    end
    final_count = result.size
  end
end`

	registry := NewRegistry()
	RegisterDefaults(registry)

	scanner := NewScanner(registry)
	symbols := scanner.Parse("/test/test.rb", []byte(content))

	// Find the method
	var method *types.Symbol
	for _, sym := range symbols {
		if sym.Kind == types.KindMethod && sym.Name == "perform" {
			method = sym
			break
		}
	}

	if method == nil {
		t.Fatal("Method 'perform' not found")
	}

	t.Logf("Method perform: lines %d-%d", method.Line, method.EndLine)

	// Method should end on line 12 (the 'end' of 'def perform')
	if method.EndLine != 12 {
		t.Errorf("Expected method EndLine to be 12, got %d", method.EndLine)
	}

	// Find all local variables
	var localVars []*types.Symbol
	for _, sym := range symbols {
		if sym.Kind == types.KindLocalVariable {
			localVars = append(localVars, sym)
			t.Logf("Local var %s at line %d in method %s", sym.Name, sym.Line, sym.MethodFullName)
		}
	}

	// Should find: items (line 3), result (line 7), final_count (line 11)
	if len(localVars) != 3 {
		t.Errorf("Expected 3 local variables, got %d", len(localVars))
	}

	// Verify all local variables are assigned to the method
	for _, lv := range localVars {
		if lv.MethodFullName != method.FullName {
			t.Errorf("Local var %s should be in method %s, but is in %s",
				lv.Name, method.FullName, lv.MethodFullName)
		}
	}
}
