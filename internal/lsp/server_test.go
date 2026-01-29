package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/index"
	"github.com/jarredhawkins/goruby-lsp/internal/parser"
)

func TestReferencesDeduplication(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "lsp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Ruby file with a method definition and call
	testContent := `class Person
  def validate_record!
    # This is the definition
    check_fields
  end

  def process
    validate_record!
  end
end
`
	testFile := filepath.Join(tmpDir, "person.rb")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create the index
	registry := parser.NewRegistry()
	parser.RegisterDefaults(registry)
	idx := index.New(tmpDir, registry)

	// Add the test file
	if err := idx.AddFile(testFile); err != nil {
		t.Fatalf("failed to add file to index: %v", err)
	}

	// Test FindReferences - this is the trigram search
	word := "validate_record!"
	refs := idx.FindReferences(word)

	t.Logf("FindReferences found %d references for %q:", len(refs), word)
	for i, ref := range refs {
		t.Logf("  [%d] %s line %d col %d: %s", i, ref.FilePath, ref.Line, ref.Column, ref.LineText)
	}

	// Test FindDefinitions
	symbols := idx.FindDefinitions(word)
	t.Logf("FindDefinitions found %d definitions for %q:", len(symbols), word)
	for i, sym := range symbols {
		t.Logf("  [%d] %s line %d col %d", i, sym.FilePath, sym.Line, sym.Column)
	}

	// Simulate what handleReferences does with deduplication
	seen := make(map[string]struct{})
	var uniqueLocations []struct {
		file string
		line int
		col  int
	}

	// Add references
	for _, ref := range refs {
		key := ref.FilePath + ":" + string(rune(ref.Line)) + ":" + string(rune(ref.Column))
		if _, exists := seen[key]; exists {
			t.Logf("Skipping duplicate reference at line %d col %d", ref.Line, ref.Column)
			continue
		}
		seen[key] = struct{}{}
		uniqueLocations = append(uniqueLocations, struct {
			file string
			line int
			col  int
		}{ref.FilePath, ref.Line, ref.Column})
	}

	// Add definitions (simulating IncludeDeclaration: true)
	for _, sym := range symbols {
		key := sym.FilePath + ":" + string(rune(sym.Line)) + ":" + string(rune(sym.Column))
		if _, exists := seen[key]; exists {
			t.Logf("Skipping duplicate definition at line %d col %d (already in references)", sym.Line, sym.Column)
			continue
		}
		seen[key] = struct{}{}
		uniqueLocations = append(uniqueLocations, struct {
			file string
			line int
			col  int
		}{sym.FilePath, sym.Line, sym.Column})
	}

	t.Logf("Final unique locations: %d", len(uniqueLocations))
	for i, loc := range uniqueLocations {
		t.Logf("  [%d] %s line %d col %d", i, loc.file, loc.line, loc.col)
	}

	// We expect exactly 2 unique locations:
	// 1. The definition on line 2 (1-indexed)
	// 2. The call on line 8 (1-indexed)
	if len(uniqueLocations) != 2 {
		t.Errorf("Expected 2 unique locations, got %d", len(uniqueLocations))
	}

	// Verify the definition was deduplicated
	if len(refs) > 0 && len(symbols) > 0 {
		// The definition line should appear in both refs and symbols
		defLine := symbols[0].Line
		defInRefs := false
		for _, ref := range refs {
			if ref.Line == defLine {
				defInRefs = true
				break
			}
		}
		if defInRefs {
			t.Logf("Definition line %d appears in both refs and symbols - deduplication is required", defLine)
		}
	}
}

func TestReferencesDeduplicationWithBetterKey(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "lsp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Ruby file with a method definition and call
	testContent := `class Person
  def validate_record!
    # This is the definition
    check_fields
  end

  def process
    validate_record!
  end
end
`
	testFile := filepath.Join(tmpDir, "person.rb")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create the index
	registry := parser.NewRegistry()
	parser.RegisterDefaults(registry)
	idx := index.New(tmpDir, registry)

	// Add the test file
	if err := idx.AddFile(testFile); err != nil {
		t.Fatalf("failed to add file to index: %v", err)
	}

	word := "validate_record!"
	refs := idx.FindReferences(word)
	symbols := idx.FindDefinitions(word)

	// Use the same key format as in server.go (fmt.Sprintf)
	seen := make(map[string]struct{})
	var uniqueCount int

	for _, ref := range refs {
		key := ref.FilePath + ":" + itoa(ref.Line) + ":" + itoa(ref.Column)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		uniqueCount++
	}

	for _, sym := range symbols {
		key := sym.FilePath + ":" + itoa(sym.Line) + ":" + itoa(sym.Column)
		if _, exists := seen[key]; exists {
			t.Logf("Definition at %s:%d:%d was deduplicated correctly", sym.FilePath, sym.Line, sym.Column)
			continue
		}
		seen[key] = struct{}{}
		uniqueCount++
	}

	if uniqueCount != 2 {
		t.Errorf("Expected 2 unique locations after deduplication, got %d", uniqueCount)
	}
}

func TestExtractWordAt_NamespacedClass(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		char     int
		expected string
	}{
		{
			name:     "cursor on EinMatcher in EinLetter::EinMatcher.new",
			line:     "    EinLetter::EinMatcher.new",
			char:     18, // on 'E' of EinMatcher
			expected: "EinLetter::EinMatcher",
		},
		{
			name:     "cursor on EinLetter in EinLetter::EinMatcher",
			line:     "    EinLetter::EinMatcher",
			char:     6, // on 'e' of EinLetter
			expected: "EinLetter",
		},
		{
			name:     "leading :: preserved",
			line:     "  ::TopLevel::Foo.call",
			char:     16, // on 'o' of Foo
			expected: "::TopLevel::Foo",
		},
		{
			name:     "triple nested",
			line:     "A::B::C.new",
			char:     6, // on 'C'
			expected: "A::B::C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWordAt(tt.line, 0, tt.char)
			if result != tt.expected {
				t.Errorf("extractWordAt(%q, 0, %d) = %q, want %q", tt.line, tt.char, result, tt.expected)
			}
		})
	}
}

// itoa converts int to string (simple helper)
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if negative {
		s = "-" + s
	}
	return s
}
