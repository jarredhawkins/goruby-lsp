package parser

import (
	"regexp"
)

// Matches Ruby block-opening keywords at the start of a line.
// These constructs require an `end` but don't create a named scope.
// We need to track them so their `end` doesn't over-decrement nesting depth.
//
// Matches: if, unless, case, while, until, for, begin
// Does NOT match postfix if/unless (e.g., "return if x" — these don't start the line)
var blockPattern = regexp.MustCompile(`^\s*(if|unless|case|while|until|for|begin)\b`)

// BlockMatcher tracks block-opening keywords that require `end`
type BlockMatcher struct{}

func (m *BlockMatcher) Name() string  { return "block" }
func (m *BlockMatcher) Priority() int { return 55 } // Above end (50), below do (60)

func (m *BlockMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	if !blockPattern.MatchString(line) {
		return nil
	}
	return &MatchResult{
		OpensBlock: true,
	}
}
