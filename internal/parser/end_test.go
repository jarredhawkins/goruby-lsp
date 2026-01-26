package parser

import (
	"testing"
)

func TestEndMatcher(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNil bool
	}{
		{
			name:    "simple end",
			line:    "end",
			wantNil: false,
		},
		{
			name:    "indented end",
			line:    "  end",
			wantNil: false,
		},
		{
			name:    "end with comment",
			line:    "end # close class",
			wantNil: false,
		},
		{
			name:    "not end keyword",
			line:    "def my_method",
			wantNil: true,
		},
		{
			name:    "end in word",
			line:    "  endpoint = '/api'",
			wantNil: true,
		},
		{
			name:    "send method",
			line:    "  obj.send(:method)",
			wantNil: true,
		},
	}

	matcher := &EndMatcher{}
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
			if !result.PopScope {
				t.Error("expected PopScope to be true")
			}
		})
	}
}
