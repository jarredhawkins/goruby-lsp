package parser

import "testing"

func TestDoMatcher(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"simple do", "items.each do", true},
		{"do with block param", "items.each do |item|", true},
		{"do with multiple params", "hash.each do |key, value|", true},
		{"do with index", "items.each_with_index do |item, idx|", true},
		{"indented do", "    items.map do |x|", true},
		{"loop do", "loop do", true},
		{"lambda do", "proc = lambda do", true},
		{"Thread.new do", "Thread.new do", true},
		{"not do - method call", "do_something", false},
		{"not do - method with args", "do_something(1, 2)", false},
		{"not do - redo keyword", "redo if condition", false},
		{"not do - middle of line", "items.each do |x| puts x end", false}, // single-line block
	}

	matcher := &DoMatcher{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &ParseContext{
				FilePath: "/test/test.rb",
				LineNum:  1,
			}

			result := matcher.Match(tc.line, ctx)

			if tc.expected && result == nil {
				t.Errorf("Expected match for %q, got nil", tc.line)
			}
			if !tc.expected && result != nil {
				t.Errorf("Expected no match for %q, got result", tc.line)
			}
			if result != nil && !result.OpensBlock {
				t.Errorf("Expected OpensBlock=true for %q", tc.line)
			}
		})
	}
}
