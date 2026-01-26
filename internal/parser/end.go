package parser

import (
	"regexp"
)

// end keyword (for scope tracking)
var endPattern = regexp.MustCompile(`^\s*end\b`)

// EndMatcher tracks scope closing
type EndMatcher struct{}

func (m *EndMatcher) Name() string  { return "end" }
func (m *EndMatcher) Priority() int { return 50 }

func (m *EndMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	if !endPattern.MatchString(line) {
		return nil
	}
	return &MatchResult{
		PopScope: true,
	}
}
