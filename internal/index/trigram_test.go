package index

import (
	"testing"
)

func TestTrigramSearchWithBangMethod(t *testing.T) {
	idx := NewTrigramIndex()

	content := `class MyClass
  def perform
    ensure_valid_latest_evaluation!(entity, token)
    do_something
    ensure_valid_latest_evaluation!(other_entity, other_token)
  end

  def ensure_valid_latest_evaluation!(entity, token)
    # implementation
  end
end`

	idx.AddFile("/test/file.rb", []byte(content))

	refs := idx.Search("ensure_valid_latest_evaluation!")

	t.Logf("Found %d references", len(refs))
	for _, ref := range refs {
		t.Logf("  Line %d, Col %d, Len %d: %s", ref.Line, ref.Column, ref.Length, ref.LineText)
	}

	// Should find 3 references: 2 calls + 1 definition
	if len(refs) != 3 {
		t.Errorf("Expected 3 references, got %d", len(refs))
	}
}

func TestTrigramSearchWithQuestionMethod(t *testing.T) {
	idx := NewTrigramIndex()

	content := `class MyClass
  def valid?
    true
  end

  def check
    if valid?
      puts "yes"
    end
  end
end`

	idx.AddFile("/test/file.rb", []byte(content))

	refs := idx.Search("valid?")

	t.Logf("Found %d references", len(refs))
	for _, ref := range refs {
		t.Logf("  Line %d, Col %d, Len %d: %s", ref.Line, ref.Column, ref.Length, ref.LineText)
	}

	// Should find 2 references: 1 def + 1 call
	if len(refs) != 2 {
		t.Errorf("Expected 2 references, got %d", len(refs))
	}
}

func TestBuildWordBoundaryPattern(t *testing.T) {
	tests := []struct {
		pattern     string
		text        string
		shouldMatch bool
	}{
		{"foo!", "foo!(bar)", true},
		{"foo!", "foo!bar", false},    // foo! followed by word char
		{"foo!", "xfoo!(bar)", false}, // no word boundary before
		{"foo?", "foo?(bar)", true},
		{"foo?", "if foo?", true},
		{"foo=", "self.foo=(val)", true},
		{"foo", "foo(bar)", true},
		{"foo", "foobar", false},
	}

	for _, tc := range tests {
		re := buildWordBoundaryPattern(tc.pattern)
		matched := re.MatchString(tc.text)
		if matched != tc.shouldMatch {
			t.Errorf("Pattern %q against %q: expected match=%v, got %v (regex: %s)",
				tc.pattern, tc.text, tc.shouldMatch, matched, re.String())
		}
	}
}
