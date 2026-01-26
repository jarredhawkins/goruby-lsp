package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// def my_method
// def my_method(args)
// def self.my_class_method
var methodPattern = regexp.MustCompile(`^\s*def\s+(self\.)?(\w+[?!=]?)`)

// MethodMatcher extracts method definitions
type MethodMatcher struct{}

func (m *MethodMatcher) Name() string  { return "method" }
func (m *MethodMatcher) Priority() int { return 90 }

func (m *MethodMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	match := methodPattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	isSingleton := match[1] != "" // self.
	methodName := match[2]

	col := strings.Index(line, methodName)

	kind := types.KindMethod
	if isSingleton {
		kind = types.KindSingletonMethod
	}

	sym := &types.Symbol{
		Name:     methodName,
		Kind:     kind,
		FilePath: ctx.FilePath,
		Line:     ctx.LineNum,
		Column:   col,
		Scope:    append([]string{}, ctx.CurrentScope...),
	}
	sym.FullName = sym.ComputeFullName()

	return &MatchResult{
		Symbols: []*types.Symbol{sym},
		EnterMethod: &MethodContext{
			FullName:  sym.FullName,
			StartLine: ctx.LineNum,
			// NestingDepth will be set by scanner
		},
	}
}
