package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// MY_CONSTANT = value
// MyConstant = value
var constantPattern = regexp.MustCompile(`^\s*([A-Z][A-Z0-9_]*)\s*=`)

// Pattern to detect comparison operators (==, ===)
var constantComparisonPattern = regexp.MustCompile(`^\s*[A-Z][A-Z0-9_]*\s*={2,3}`)

// ConstantMatcher extracts constant definitions
type ConstantMatcher struct{}

func (m *ConstantMatcher) Name() string  { return "constant" }
func (m *ConstantMatcher) Priority() int { return 80 }

func (m *ConstantMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	// Skip comparison operators (==, ===)
	if constantComparisonPattern.MatchString(line) {
		return nil
	}

	match := constantPattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	constName := match[1]
	col := strings.Index(line, constName)

	sym := &types.Symbol{
		Name:     constName,
		Kind:     types.KindConstant,
		FilePath: ctx.FilePath,
		Line:     ctx.LineNum,
		Column:   col,
		Scope:    append([]string{}, ctx.CurrentScope...),
	}
	sym.FullName = sym.ComputeFullName()

	return &MatchResult{
		Symbols: []*types.Symbol{sym},
	}
}
