package parser

import (
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// Scanner parses Ruby files line by line
type Scanner struct {
	registry *Registry
}

// NewScanner creates a new scanner with the given registry
func NewScanner(registry *Registry) *Scanner {
	return &Scanner{
		registry: registry,
	}
}

// tryStartMultiline checks if any matcher wants to start multi-line accumulation
func (s *Scanner) tryStartMultiline(matchers []Matcher, line string, lineNum int) *accumulator {
	for _, matcher := range matchers {
		if detector, ok := matcher.(MultilineDetector); ok {
			if isStart, opener, closer := detector.StartsMultiline(line); isStart {
				acc := &accumulator{
					startLine: lineNum,
					opener:    opener,
					closer:    closer,
				}
				acc.addLine(line)
				return acc
			}
		}
	}
	return nil
}

// accumulator tracks multi-line construct state
type accumulator struct {
	buffer    strings.Builder
	startLine int
	opener    string
	closer    string
	depth     int
}

func (a *accumulator) addLine(line string) {
	if a.buffer.Len() > 0 {
		a.buffer.WriteString(" ")
	}
	a.buffer.WriteString(line)
	a.depth += strings.Count(line, a.opener) - strings.Count(line, a.closer)
}

func (a *accumulator) isComplete() bool {
	return a.depth <= 0
}

func (a *accumulator) content() string {
	return a.buffer.String()
}

// scanState holds the shared scope/nesting state during a scan.
type scanState struct {
	ScopeStack   []string
	NestingDepth int
}

// scanCallbacks controls the scan loop behavior.
type scanCallbacks struct {
	// beforeMatch is called at the start of each line, before matchers run.
	// Use it to set up context (e.g. CurrentMethod). May be nil.
	beforeMatch func(ctx *ParseContext, state *scanState)

	// onResult is called after a matcher produces a result, before scope/nesting
	// updates are applied. Return false to stop scanning.
	onResult func(ctx *ParseContext, result *MatchResult, state *scanState) bool
}

// scanLines runs the core line-by-line parse loop.
func (s *Scanner) scanLines(content []byte, filePath string, cb scanCallbacks) *scanState {
	lines := strings.Split(string(content), "\n")
	state := &scanState{}

	ctx := &ParseContext{
		FilePath:     filePath,
		CurrentScope: state.ScopeStack,
	}

	matchers := s.registry.Matchers()
	var acc *accumulator

	for lineNum, line := range lines {
		ctx.LineNum = lineNum + 1
		ctx.CurrentScope = state.ScopeStack

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if acc != nil {
			acc.addLine(trimmed)
			if !acc.isComplete() {
				continue
			}
			ctx.LineNum = acc.startLine
			line = acc.content()
			acc = nil
		} else if acc = s.tryStartMultiline(matchers, trimmed, ctx.LineNum); acc != nil {
			if !acc.isComplete() {
				continue
			}
			line = acc.content()
			acc = nil
		}

		if cb.beforeMatch != nil {
			cb.beforeMatch(ctx, state)
		}

		for _, matcher := range matchers {
			result := matcher.Match(line, ctx)
			if result == nil {
				continue
			}

			if !cb.onResult(ctx, result, state) {
				return state
			}

			if result.PushScope != "" {
				state.ScopeStack = append(state.ScopeStack, result.PushScope)
			}
			if result.OpensBlock {
				state.NestingDepth++
			}
			if result.ClosesBlock && state.NestingDepth > 0 {
				state.NestingDepth--
			}
			if result.PopScope && state.NestingDepth < len(state.ScopeStack) {
				state.ScopeStack = state.ScopeStack[:len(state.ScopeStack)-1]
			}
			break
		}
	}

	return state
}

// Parse scans the file content and returns all discovered symbols
func (s *Scanner) Parse(filePath string, content []byte) []*types.Symbol {
	var symbols []*types.Symbol
	var currentMethod *MethodContext
	var methodSymbol *types.Symbol

	s.scanLines(content, filePath, scanCallbacks{
		beforeMatch: func(ctx *ParseContext, state *scanState) {
			ctx.CurrentMethod = currentMethod
		},
		onResult: func(ctx *ParseContext, result *MatchResult, state *scanState) bool {
			symbols = append(symbols, result.Symbols...)

			if result.EnterMethod != nil {
				currentMethod = result.EnterMethod
				// NestingDepth will be incremented after this callback returns,
				// so add 1 to account for the block this result opens.
				currentMethod.NestingDepth = state.NestingDepth + 1
				for _, sym := range result.Symbols {
					if sym.Kind == types.KindMethod || sym.Kind == types.KindSingletonMethod {
						methodSymbol = sym
						break
					}
				}
			}

			if result.ClosesBlock && state.NestingDepth > 0 {
				// Check BEFORE scanLines decrements nesting
				if currentMethod != nil && state.NestingDepth == currentMethod.NestingDepth {
					if methodSymbol != nil {
						methodSymbol.EndLine = ctx.LineNum
						methodSymbol = nil
					}
					currentMethod = nil
				}
			}

			return true
		},
	})

	return symbols
}

// ScopeAtLine returns the scope stack at the given 1-indexed line.
func (s *Scanner) ScopeAtLine(content []byte, targetLine int) []string {
	state := s.scanLines(content, "", scanCallbacks{
		onResult: func(ctx *ParseContext, result *MatchResult, state *scanState) bool {
			return ctx.LineNum <= targetLine
		},
	})

	scope := make([]string, len(state.ScopeStack))
	copy(scope, state.ScopeStack)
	return scope
}

// ParseFile reads and parses a Ruby file
func (s *Scanner) ParseFile(filePath string) ([]*types.Symbol, error) {
	// This would read the file, but we'll let the index handle file reading
	// to avoid circular imports and for better caching
	return nil, nil
}
