package parser

import (
	"regexp"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// RelationMatcher extracts Rails relation definitions (belongs_to, has_one, has_many)
type RelationMatcher struct{}

func (m *RelationMatcher) Name() string  { return "relation" }
func (m *RelationMatcher) Priority() int { return 85 }

// Pattern: belongs_to/has_one/has_many :name, optional class_name: 'ClassName'
// Updated to handle whitespace between elements for multi-line support
var relationPattern = regexp.MustCompile(
	`^\s*(belongs_to|has_one|has_many)\s*[\(\s]+:([a-z_][a-z0-9_]*)` +
		`(?:.*class_name:\s*['"]([A-Za-z][A-Za-z0-9_:]*)['"'])?`,
)

// multilineStartPattern detects start of multi-line relation definitions
var multilineStartPattern = regexp.MustCompile(`^\s*(belongs_to|has_one|has_many)\s*\(`)

// StartsMultiline implements MultilineDetector
func (m *RelationMatcher) StartsMultiline(line string) (bool, string, string) {
	if !multilineStartPattern.MatchString(line) {
		return false, "", ""
	}
	// Check if line has unclosed parens
	openCount := strings.Count(line, "(")
	closeCount := strings.Count(line, ")")
	if openCount > closeCount {
		return true, "(", ")"
	}
	return false, "", ""
}

func (m *RelationMatcher) Match(line string, ctx *ParseContext) *MatchResult {
	// Only match inside classes
	if len(ctx.CurrentScope) == 0 {
		return nil
	}

	match := relationPattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	relationType := match[1] // belongs_to, has_one, has_many
	relationName := match[2] // :address → address
	className := match[3]    // optional class_name: 'Person'

	// Resolve target class name
	var targetClass string
	if className != "" {
		targetClass = className
	} else {
		// Infer from relation name
		targetClass = toClassName(relationName, relationType == "has_many")
	}

	col := strings.Index(line, ":"+relationName) + 1 // Position of relation symbol

	sym := &types.Symbol{
		Name:       relationName,
		TargetName: targetClass,
		Kind:       types.KindRelation,
		FilePath:   ctx.FilePath,
		Line:       ctx.LineNum,
		Column:     col,
		Scope:      append([]string{}, ctx.CurrentScope...),
	}
	sym.FullName = sym.ComputeFullName()

	return &MatchResult{Symbols: []*types.Symbol{sym}}
}

// toClassName converts snake_case to CamelCase, with optional singularization
func toClassName(name string, singularize bool) string {
	// Convert snake_case to CamelCase
	parts := strings.Split(name, "_")

	// Singularize only the last part (e.g., business_people → business_person)
	if singularize && len(parts) > 0 {
		parts[len(parts)-1] = singular(parts[len(parts)-1])
	}

	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// singular handles common English pluralization rules
func singular(word string) string {
	// Handle common irregular plurals
	irregulars := map[string]string{
		"people": "person", "children": "child", "men": "man",
		"women": "woman", "teeth": "tooth", "feet": "foot",
		"mice": "mouse", "geese": "goose",
	}
	if s, ok := irregulars[word]; ok {
		return s
	}

	// Handle common patterns
	if strings.HasSuffix(word, "ies") && len(word) > 3 {
		return word[:len(word)-3] + "y" // companies → company
	}
	if strings.HasSuffix(word, "ves") && len(word) > 3 {
		return word[:len(word)-3] + "f" // leaves → leaf
	}
	if strings.HasSuffix(word, "ses") || strings.HasSuffix(word, "xes") ||
		strings.HasSuffix(word, "zes") || strings.HasSuffix(word, "ches") ||
		strings.HasSuffix(word, "shes") {
		return word[:len(word)-2] // boxes → box, watches → watch
	}
	if strings.HasSuffix(word, "s") && len(word) > 1 {
		return word[:len(word)-1] // comments → comment
	}
	return word
}
