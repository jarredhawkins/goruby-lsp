package parser

import (
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func TestRelationMatcher(t *testing.T) {
	matcher := &RelationMatcher{}

	tests := []struct {
		name           string
		line           string
		scope          []string
		wantMatch      bool
		wantName       string
		wantTargetName string
	}{
		{
			name:           "simple belongs_to",
			line:           "  belongs_to :address",
			scope:          []string{"User"},
			wantMatch:      true,
			wantName:       "address",
			wantTargetName: "Address",
		},
		{
			name:           "belongs_to with class_name",
			line:           "  belongs_to :owner, class_name: 'Person'",
			scope:          []string{"User"},
			wantMatch:      true,
			wantName:       "owner",
			wantTargetName: "Person",
		},
		{
			name:           "belongs_to with double quotes",
			line:           `  belongs_to :owner, class_name: "Person"`,
			scope:          []string{"User"},
			wantMatch:      true,
			wantName:       "owner",
			wantTargetName: "Person",
		},
		{
			name:           "has_one simple",
			line:           "  has_one :business_structure",
			scope:          []string{"Company"},
			wantMatch:      true,
			wantName:       "business_structure",
			wantTargetName: "BusinessStructure",
		},
		{
			name:           "has_many singularizes",
			line:           "  has_many :comments",
			scope:          []string{"Post"},
			wantMatch:      true,
			wantName:       "comments",
			wantTargetName: "Comment",
		},
		{
			name:           "has_many with class_name",
			line:           "  has_many :posts, class_name: 'Article'",
			scope:          []string{"User"},
			wantMatch:      true,
			wantName:       "posts",
			wantTargetName: "Article",
		},
		{
			name:           "namespaced class_name",
			line:           "  belongs_to :user, class_name: 'Spanner::CheckbookUser'",
			scope:          []string{"Account"},
			wantMatch:      true,
			wantName:       "user",
			wantTargetName: "Spanner::CheckbookUser",
		},
		{
			name:      "no match outside class",
			line:      "  belongs_to :address",
			scope:     []string{},
			wantMatch: false,
		},
		{
			name:      "no match for non-relation",
			line:      "  validates :email",
			scope:     []string{"User"},
			wantMatch: false,
		},
		{
			name:           "has_many with irregular plural",
			line:           "  has_many :people",
			scope:          []string{"Company"},
			wantMatch:      true,
			wantName:       "people",
			wantTargetName: "Person",
		},
		{
			name:           "has_many with -ies plural",
			line:           "  has_many :companies",
			scope:          []string{"User"},
			wantMatch:      true,
			wantName:       "companies",
			wantTargetName: "Company",
		},
		{
			name:           "has_many with -es plural",
			line:           "  has_many :boxes",
			scope:          []string{"Warehouse"},
			wantMatch:      true,
			wantName:       "boxes",
			wantTargetName: "Box",
		},
		{
			name:           "relation with other options",
			line:           "  belongs_to :author, class_name: 'User', foreign_key: :user_id",
			scope:          []string{"Post"},
			wantMatch:      true,
			wantName:       "author",
			wantTargetName: "User",
		},
		{
			name:           "has_many compound with irregular plural",
			line:           "  has_many :business_people",
			scope:          []string{"Company"},
			wantMatch:      true,
			wantName:       "business_people",
			wantTargetName: "BusinessPerson",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ParseContext{
				FilePath:     "/test/model.rb",
				CurrentScope: tt.scope,
				LineNum:      10,
			}

			result := matcher.Match(tt.line, ctx)

			if tt.wantMatch {
				if result == nil {
					t.Fatalf("expected match, got nil")
				}
				if len(result.Symbols) != 1 {
					t.Fatalf("expected 1 symbol, got %d", len(result.Symbols))
				}

				sym := result.Symbols[0]
				if sym.Name != tt.wantName {
					t.Errorf("expected Name %q, got %q", tt.wantName, sym.Name)
				}
				if sym.TargetName != tt.wantTargetName {
					t.Errorf("expected TargetName %q, got %q", tt.wantTargetName, sym.TargetName)
				}
				if sym.Kind != types.KindRelation {
					t.Errorf("expected Kind %v, got %v", types.KindRelation, sym.Kind)
				}
			} else {
				if result != nil {
					t.Errorf("expected no match, got %+v", result)
				}
			}
		})
	}
}

func TestSingular(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"comments", "comment"},
		{"posts", "post"},
		{"companies", "company"},
		{"boxes", "box"},
		{"watches", "watch"},
		{"addresses", "address"}, // -es ending handled
		{"people", "person"},
		{"children", "child"},
		{"leaves", "leaf"},
		{"mice", "mouse"},
		{"user", "user"}, // Already singular
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := singular(tt.input)
			if result != tt.expected {
				t.Errorf("singular(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRelationMatcher_MultiLine(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTarget string
		wantName   string
	}{
		{
			name: "multi-line has_many with class_name",
			input: `class Order
  has_many(
    :items,
    class_name: 'LineItem',
    foreign_key: :order_id,
  )
end`,
			wantTarget: "LineItem",
			wantName:   "items",
		},
		{
			name: "multi-line belongs_to with namespaced class_name",
			input: `class Comment
  belongs_to(
    :author,
    class_name: 'Admin::User',
  )
end`,
			wantTarget: "Admin::User",
			wantName:   "author",
		},
		{
			name: "multi-line has_one without class_name",
			input: `class User
  has_one(
    :profile,
    dependent: :destroy,
  )
end`,
			wantTarget: "Profile",
			wantName:   "profile",
		},
		{
			name: "multi-line has_many infers singular class",
			input: `class Post
  has_many(
    :comments,
    dependent: :destroy,
  )
end`,
			wantTarget: "Comment",
			wantName:   "comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			RegisterDefaults(registry)
			scanner := NewScanner(registry)

			symbols := scanner.Parse("/test/model.rb", []byte(tt.input))

			// Find the relation symbol
			var relationSym *types.Symbol
			for _, sym := range symbols {
				if sym.Kind == types.KindRelation {
					relationSym = sym
					break
				}
			}

			if relationSym == nil {
				t.Fatalf("expected to find a relation symbol, got none. All symbols: %+v", symbols)
			}

			if relationSym.Name != tt.wantName {
				t.Errorf("expected Name %q, got %q", tt.wantName, relationSym.Name)
			}
			if relationSym.TargetName != tt.wantTarget {
				t.Errorf("expected TargetName %q, got %q", tt.wantTarget, relationSym.TargetName)
			}
		})
	}
}

func TestToClassName(t *testing.T) {
	tests := []struct {
		name        string
		singularize bool
		expected    string
	}{
		{"address", false, "Address"},
		{"business_structure", false, "BusinessStructure"},
		{"comments", true, "Comment"},
		{"user_profiles", true, "UserProfile"},
		{"person", false, "Person"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toClassName(tt.name, tt.singularize)
			if result != tt.expected {
				t.Errorf("toClassName(%q, %v) = %q, want %q", tt.name, tt.singularize, result, tt.expected)
			}
		})
	}
}
