package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// module MyModule
// module MyParent::MyModule
var modulePattern = regexp.MustCompile(`^\s*module\s+([A-Z]\w*(?:::[A-Z]\w*)*)`)

// ModuleMatcher extracts module definitions
type ModuleMatcher struct{}

func (m *ModuleMatcher) Name() string  { return "module" }
func (m *ModuleMatcher) Priority() int { return 100 }

func (m *ModuleMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	match := modulePattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	moduleName := match[1]
	col := strings.Index(line, moduleName)

	// Handle nested module names
	parts := strings.Split(moduleName, "::")
	shortName := parts[len(parts)-1]

	scope := append([]string{}, ctx.CurrentScope...)
	if len(parts) > 1 {
		scope = append(scope, parts[:len(parts)-1]...)
	}

	sym := &types.Symbol{
		Name:     shortName,
		Kind:     types.KindModule,
		FilePath: ctx.FilePath,
		Line:     ctx.LineNum,
		Column:   col,
		Scope:    scope,
	}
	sym.FullName = sym.ComputeFullName()

	return &MatchResult{
		Symbols:    []*types.Symbol{sym},
		PushScope:  shortName,
		OpensBlock: true,
	}
}
