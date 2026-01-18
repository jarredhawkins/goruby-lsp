package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// Local variable patterns
var (
	// Single assignment: x = 1
	// We match the pattern and check in code that it's not == or ===
	singleAssignPattern = regexp.MustCompile(`^\s*([a-z_][a-z0-9_]*)\s*=`)

	// Multiple assignment: x, y = 1, 2
	multiAssignPattern = regexp.MustCompile(`^\s*([a-z_][a-z0-9_]*(?:\s*,\s*[a-z_][a-z0-9_]*)+)\s*=`)

	// Pattern to detect comparison operators (==, ===, =~)
	comparisonPattern = regexp.MustCompile(`^\s*[a-z_][a-z0-9_]*\s*(?:={2,3}|=~)`)
)

// LocalVariableMatcher extracts local variable assignments inside methods
type LocalVariableMatcher struct{}

func (m *LocalVariableMatcher) Name() string  { return "localvar" }
func (m *LocalVariableMatcher) Priority() int { return 70 } // Below constants (80)

func (m *LocalVariableMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	// Only match inside methods
	if ctx.CurrentMethod == nil {
		return nil
	}

	// Skip comparison operators (==, ===, =~)
	if comparisonPattern.MatchString(line) {
		return nil
	}

	// Try multiple assignment first (more specific pattern)
	if match := multiAssignPattern.FindStringSubmatch(line); match != nil {
		return m.handleMultiAssign(match[1], line, ctx)
	}

	// Try single assignment
	if match := singleAssignPattern.FindStringSubmatch(line); match != nil {
		return m.handleSingleAssign(match[1], line, ctx)
	}

	return nil
}

func (m *LocalVariableMatcher) handleSingleAssign(varName, line string, ctx *ParseContext) *MatchResult {
	col := strings.Index(line, varName)

	sym := &types.Symbol{
		Name:           varName,
		Kind:           types.KindLocalVariable,
		FilePath:       ctx.FilePath,
		Line:           ctx.LineNum,
		Column:         col,
		Scope:          append([]string{}, ctx.CurrentScope...),
		MethodFullName: ctx.CurrentMethod.FullName,
	}
	sym.FullName = sym.ComputeFullName()

	return &MatchResult{
		Symbols: []*types.Symbol{sym},
	}
}

func (m *LocalVariableMatcher) handleMultiAssign(varList, line string, ctx *ParseContext) *MatchResult {
	// Parse comma-separated variable names
	vars := strings.Split(varList, ",")
	var symbols []*types.Symbol

	for _, v := range vars {
		varName := strings.TrimSpace(v)
		if varName == "" {
			continue
		}

		col := strings.Index(line, varName)

		sym := &types.Symbol{
			Name:           varName,
			Kind:           types.KindLocalVariable,
			FilePath:       ctx.FilePath,
			Line:           ctx.LineNum,
			Column:         col,
			Scope:          append([]string{}, ctx.CurrentScope...),
			MethodFullName: ctx.CurrentMethod.FullName,
		}
		sym.FullName = sym.ComputeFullName()

		symbols = append(symbols, sym)
	}

	if len(symbols) == 0 {
		return nil
	}

	return &MatchResult{
		Symbols: symbols,
	}
}
