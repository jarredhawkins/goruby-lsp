package parser

import (
	"regexp"
)

// do |x| or do at end of line (block start)
// Matches: foo.each do |x|, loop do, etc.
var doPattern = regexp.MustCompile(`\bdo\s*(\|[^|]*\|)?\s*$`)

// DoMatcher tracks do...end block nesting
type DoMatcher struct{}

func (m *DoMatcher) Name() string  { return "do" }
func (m *DoMatcher) Priority() int { return 60 } // Below local vars (70), above end (50)

func (m *DoMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	if !doPattern.MatchString(line) {
		return nil
	}
	// Opens a block but doesn't create a named scope
	return &MatchResult{
		OpensBlock: true,
	}
}
