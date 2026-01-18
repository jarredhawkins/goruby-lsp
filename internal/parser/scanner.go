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

// Parse scans the file content and returns all discovered symbols
func (s *Scanner) Parse(filePath string, content []byte) []*types.Symbol {
	lines := strings.Split(string(content), "\n")
	var symbols []*types.Symbol
	var scopeStack []string
	var nestingDepth int             // Track all block nesting (class, module, method, etc.)
	var currentMethod *MethodContext // Current method being parsed
	var methodSymbol *types.Symbol   // Symbol for current method (to set EndLine)

	ctx := &ParseContext{
		FilePath:     filePath,
		CurrentScope: scopeStack,
	}

	matchers := s.registry.Matchers()

	for lineNum, line := range lines {
		ctx.LineNum = lineNum + 1 // 1-indexed
		ctx.CurrentScope = scopeStack
		ctx.CurrentMethod = currentMethod

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Try each matcher in priority order
		for _, matcher := range matchers {
			result := matcher.Match(line, ctx)
			if result == nil {
				continue
			}

			// Collect symbols
			symbols = append(symbols, result.Symbols...)

			// Handle scope changes (class/module)
			if result.PushScope != "" {
				scopeStack = append(scopeStack, result.PushScope)
				nestingDepth++
			}

			// Handle method entry
			if result.EnterMethod != nil {
				nestingDepth++
				currentMethod = result.EnterMethod
				currentMethod.NestingDepth = nestingDepth
				// Track the method symbol to set EndLine later
				for _, sym := range result.Symbols {
					if sym.Kind == types.KindMethod || sym.Kind == types.KindSingletonMethod {
						methodSymbol = sym
						break
					}
				}
			}

			// Handle scope exit (end keyword)
			if result.PopScope {
				if nestingDepth > 0 {
					// Check if we're exiting a method
					if currentMethod != nil && nestingDepth == currentMethod.NestingDepth {
						// Method ended - set EndLine on the method symbol
						if methodSymbol != nil {
							methodSymbol.EndLine = ctx.LineNum
							methodSymbol = nil
						}
						currentMethod = nil
					}
					nestingDepth--
					// Only pop from scopeStack if nesting matches scope count
					if nestingDepth < len(scopeStack) {
						scopeStack = scopeStack[:len(scopeStack)-1]
					}
				}
			}

			// Only first match per line
			break
		}
	}

	return symbols
}

// ParseFile reads and parses a Ruby file
func (s *Scanner) ParseFile(filePath string) ([]*types.Symbol, error) {
	// This would read the file, but we'll let the index handle file reading
	// to avoid circular imports and for better caching
	return nil, nil
}
