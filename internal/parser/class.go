package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// class MyClass < BaseClass
// class MyModule::MyClass
var classPattern = regexp.MustCompile(`^\s*class\s+([A-Z]\w*(?:::[A-Z]\w*)*)(?:\s*<\s*\S+)?`)

// ClassMatcher extracts class definitions
type ClassMatcher struct{}

func (m *ClassMatcher) Name() string  { return "class" }
func (m *ClassMatcher) Priority() int { return 100 }

func (m *ClassMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	match := classPattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	className := match[1]
	col := strings.Index(line, className)

	// Handle nested class names like MyModule::MyClass
	parts := strings.Split(className, "::")
	shortName := parts[len(parts)-1]

	// Build scope: current scope + any inline modules
	scope := append([]string{}, ctx.CurrentScope...)
	if len(parts) > 1 {
		scope = append(scope, parts[:len(parts)-1]...)
	}

	sym := &types.Symbol{
		Name:     shortName,
		Kind:     types.KindClass,
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
