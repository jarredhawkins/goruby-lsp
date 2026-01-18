package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// Core Ruby patterns
var (
	// class MyClass < BaseClass
	// class MyModule::MyClass
	classPattern = regexp.MustCompile(`^\s*class\s+([A-Z]\w*(?:::[A-Z]\w*)*)(?:\s*<\s*\S+)?`)

	// module MyModule
	// module MyParent::MyModule
	modulePattern = regexp.MustCompile(`^\s*module\s+([A-Z]\w*(?:::[A-Z]\w*)*)`)

	// def my_method
	// def my_method(args)
	// def self.my_class_method
	methodPattern = regexp.MustCompile(`^\s*def\s+(self\.)?(\w+[?!=]?)`)

	// MY_CONSTANT = value
	// MyConstant = value
	constantPattern = regexp.MustCompile(`^\s*([A-Z][A-Z0-9_]*)\s*=`)

	// end keyword (for scope tracking)
	endPattern = regexp.MustCompile(`^\s*end\b`)
)

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
		Symbols:   []*types.Symbol{sym},
		PushScope: shortName,
	}
}

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
		Symbols:   []*types.Symbol{sym},
		PushScope: shortName,
	}
}

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
		Symbols:   []*types.Symbol{sym},
		PushScope: "", // Methods don't create scope for our purposes
	}
}

// ConstantMatcher extracts constant definitions
type ConstantMatcher struct{}

func (m *ConstantMatcher) Name() string  { return "constant" }
func (m *ConstantMatcher) Priority() int { return 80 }

func (m *ConstantMatcher) Match(line string, ctx *ParseContext) *MatchResult {
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
