package parser

import (
	"sort"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// MethodContext tracks the current method being parsed
type MethodContext struct {
	FullName     string // "MyClass#method_name"
	StartLine    int    // Line where method definition starts
	NestingDepth int    // Nesting depth when method started (set by scanner)
}

// ParseContext provides context for matching
type ParseContext struct {
	FilePath      string         // Absolute path of the file being parsed
	CurrentScope  []string       // Current namespace stack ["MyModule", "MyClass"]
	LineNum       int            // Current line number (1-indexed)
	CurrentMethod *MethodContext // Current method being parsed (nil if not in a method)
}

// MatchResult contains extracted symbol info from a match
type MatchResult struct {
	Symbols []*types.Symbol
	// PushScope indicates this match opens a new scope (e.g., class, module)
	PushScope string
	// PopScope indicates this match closes a scope (e.g., end keyword)
	PopScope bool
	// EnterMethod indicates this match starts a method (set by MethodMatcher)
	EnterMethod *MethodContext
}

// Matcher defines how to recognize a Ruby pattern
type Matcher interface {
	// Name returns plugin identifier
	Name() string

	// Match tests if line matches this pattern
	// Returns nil if no match
	Match(line string, ctx *ParseContext) *MatchResult

	// Priority for ordering (higher = earlier)
	Priority() int
}

// MultilineDetector is optionally implemented by matchers that handle multi-line constructs
type MultilineDetector interface {
	// StartsMultiline returns true if the line starts an incomplete multi-line construct
	// Returns (isStart, opener, closer) where opener/closer are the delimiter pair to track
	StartsMultiline(line string) (bool, string, string)
}

// Registry holds all registered matchers
type Registry struct {
	matchers []Matcher
	sorted   bool
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		matchers: make([]Matcher, 0),
	}
}

// Register adds a matcher to the registry
func (r *Registry) Register(m Matcher) {
	r.matchers = append(r.matchers, m)
	r.sorted = false
}

// Matchers returns all registered matchers in priority order
func (r *Registry) Matchers() []Matcher {
	if !r.sorted {
		sort.Slice(r.matchers, func(i, j int) bool {
			return r.matchers[i].Priority() > r.matchers[j].Priority()
		})
		r.sorted = true
	}
	return r.matchers
}

// RegisterDefaults adds the default Ruby matchers to the registry
func RegisterDefaults(r *Registry) {
	r.Register(&ClassMatcher{})
	r.Register(&ModuleMatcher{})
	r.Register(&MethodMatcher{})
	r.Register(&ConstantMatcher{})
	r.Register(&LocalVariableMatcher{})
	r.Register(&RelationMatcher{})
	r.Register(&EndMatcher{})
}
