package types

import "strings"

// SymbolKind categorizes Ruby symbols
type SymbolKind int

const (
	KindClass SymbolKind = iota
	KindModule
	KindMethod
	KindSingletonMethod
	KindConstant
	KindAttrReader
	KindAttrWriter
	KindAttrAccessor
	KindLocalVariable // Local variable inside a method
	KindCustom        // For plugin-defined symbols
	KindRelation      // Rails relation (belongs_to, has_one, has_many)
)

func (k SymbolKind) String() string {
	switch k {
	case KindClass:
		return "class"
	case KindModule:
		return "module"
	case KindMethod:
		return "method"
	case KindSingletonMethod:
		return "singleton_method"
	case KindConstant:
		return "constant"
	case KindAttrReader:
		return "attr_reader"
	case KindAttrWriter:
		return "attr_writer"
	case KindAttrAccessor:
		return "attr_accessor"
	case KindLocalVariable:
		return "local_variable"
	case KindCustom:
		return "custom"
	case KindRelation:
		return "relation"
	default:
		return "unknown"
	}
}

// Symbol represents a Ruby definition
type Symbol struct {
	Name           string // e.g., "MyClass", "my_method"
	Kind           SymbolKind
	FilePath       string // Absolute path
	Line           int    // 1-indexed
	Column         int    // 0-indexed
	EndLine        int    // For range-based symbols
	EndColumn      int
	Scope          []string // Enclosing namespaces ["MyModule", "MyClass"]
	FullName       string   // Computed: "MyModule::MyClass#my_method"
	MethodFullName string   // For local variables: the containing method's FullName
	TargetName     string   // For relations: the target class name to look up
}

// ComputeFullName generates the fully qualified name for this symbol
func (s *Symbol) ComputeFullName() string {
	var parts []string
	parts = append(parts, s.Scope...)

	switch s.Kind {
	case KindMethod, KindAttrReader, KindAttrWriter, KindAttrAccessor:
		// Instance methods use #
		if len(parts) > 0 {
			return strings.Join(parts, "::") + "#" + s.Name
		}
		return "#" + s.Name
	case KindSingletonMethod:
		// Class methods use .
		if len(parts) > 0 {
			return strings.Join(parts, "::") + "." + s.Name
		}
		return "." + s.Name
	case KindLocalVariable:
		// Local variables use @ after the method name: "MyClass#method@varname"
		if s.MethodFullName != "" {
			return s.MethodFullName + "@" + s.Name
		}
		return "@" + s.Name
	default:
		// Classes, modules, constants use ::
		parts = append(parts, s.Name)
		return strings.Join(parts, "::")
	}
}

// Reference represents a usage of a symbol
type Reference struct {
	FilePath string
	Line     int    // 1-indexed
	Column   int    // 0-indexed
	Length   int    // Length of the matched text
	LineText string // Full line text for display
}

// Location returns a simple file:line representation
func (s *Symbol) Location() string {
	return s.FilePath + ":" + string(rune('0'+s.Line))
}

// MatchesName checks if this symbol matches the given name
// Supports both short names and fully qualified names
func (s *Symbol) MatchesName(name string) bool {
	if s.Name == name {
		return true
	}
	if s.FullName == name {
		return true
	}
	return false
}
