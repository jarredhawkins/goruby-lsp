package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jarredhawkins/goruby-lsp/internal/parser"
	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

func newTestIndex() *Index {
	registry := parser.NewRegistry()
	parser.RegisterDefaults(registry)
	return New("/test", registry)
}

// addContent parses content and adds symbols to the index (test helper)
func (idx *Index) addContent(path string, content string) {
	symbols := idx.scanner.Parse(path, []byte(content))
	idx.byFile[path] = symbols
	for _, sym := range symbols {
		idx.symbols[sym.FullName] = append(idx.symbols[sym.FullName], sym)
		idx.shortNames[sym.Name] = append(idx.shortNames[sym.Name], sym.FullName)
	}
}

func TestFindDefinitions_RelationRedirect(t *testing.T) {
	idx := newTestIndex()
	idx.addContent("/test/line_item.rb", `class LineItem
end`)
	idx.addContent("/test/order.rb", `class Order
  has_many :items, class_name: 'LineItem'
end`)

	results := idx.FindDefinitions("items")
	if len(results) != 1 || results[0].Name != "LineItem" {
		t.Errorf("expected LineItem class, got %+v", results)
	}
}

func TestFindDefinitions_MultilineRelationRedirect(t *testing.T) {
	idx := newTestIndex()
	idx.addContent("/test/invoice.rb", `module Billing
  class Invoice
  end
end`)
	idx.addContent("/test/account.rb", `class Account
  has_many(
    :invoices,
    class_name: 'Billing::Invoice',
  )
end`)

	results := idx.FindDefinitions("invoices")
	if len(results) != 1 || results[0].FullName != "Billing::Invoice" {
		t.Errorf("expected Billing::Invoice class, got %+v", results)
	}
}

func TestFindDefinitions_BelongsToMultilineRedirect(t *testing.T) {
	idx := newTestIndex()
	idx.addContent("/test/parent.rb", `module Storage
  class ParentRecord
  end
end`)
	idx.addContent("/test/child.rb", `class ChildRecord
  belongs_to(
    :parent_record,
    class_name: 'Storage::ParentRecord',
  )
end`)

	results := idx.FindDefinitions("parent_record")
	if len(results) != 1 || results[0].FullName != "Storage::ParentRecord" {
		t.Errorf("expected Storage::ParentRecord class, got %+v", results)
	}
}

func TestFindDefinitions_RelationInfersTarget(t *testing.T) {
	idx := newTestIndex()
	idx.addContent("/test/comment.rb", `class Comment
end`)
	idx.addContent("/test/post.rb", `class Post
  has_many :comments
end`)

	results := idx.FindDefinitions("comments")
	if len(results) != 1 || results[0].Name != "Comment" {
		t.Errorf("expected Comment class, got %+v", results)
	}
}

func TestFindDefinitions_PartiallyQualifiedName(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "index-test-*")
	defer os.RemoveAll(tmpDir)

	// File defining the target class
	defFile := filepath.Join(tmpDir, "checker.rb")
	os.WriteFile(defFile, []byte(`module Verification
  module Matcher
    class Checker
    end
  end
end`), 0644)

	// File referencing with partial qualifier inside Verification scope
	refFile := filepath.Join(tmpDir, "runner.rb")
	os.WriteFile(refFile, []byte(`module Verification
  class Runner
    def run
      Matcher::Checker.new
    end
  end
end`), 0644)

	registry := parser.NewRegistry()
	parser.RegisterDefaults(registry)
	idx := New(tmpDir, registry)
	idx.AddFile(defFile)
	idx.AddFile(refFile)

	// Partial qualifier "Matcher::Checker" from inside Verification (line 4, 1-indexed)
	results := idx.FindDefinitionsInContext("Matcher::Checker", refFile, 4)
	if len(results) != 1 || results[0].FullName != "Verification::Matcher::Checker" {
		t.Errorf("expected Verification::Matcher::Checker, got %+v", results)
	}

	// Absolute lookup for ::Matcher::Checker should find nothing (no global Matcher::Checker)
	results = idx.FindDefinitionsInContext("::Matcher::Checker", refFile, 4)
	if len(results) != 0 {
		t.Errorf("expected no results for absolute ::Matcher::Checker, got %+v", results)
	}

	// Unqualified "Checker" should still resolve via short name
	results = idx.FindDefinitionsInContext("Checker", refFile, 4)
	if len(results) != 1 || results[0].FullName != "Verification::Matcher::Checker" {
		t.Errorf("expected Verification::Matcher::Checker via short name, got %+v", results)
	}
}

func TestFindDefinitions_NestedModuleRelation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "index-test-*")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "parent.rb"), []byte(`module Storage
  class ParentRecord
  end
end`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "child.rb"), []byte(`module Storage
  class ChildRecord
    belongs_to(
      :parent_record,
      class_name: 'Storage::ParentRecord',
    )
  end
end`), 0644)

	registry := parser.NewRegistry()
	parser.RegisterDefaults(registry)
	idx := New(tmpDir, registry)
	idx.AddFile(filepath.Join(tmpDir, "parent.rb"))
	idx.AddFile(filepath.Join(tmpDir, "child.rb"))

	results := idx.FindDefinitions("parent_record")
	if len(results) != 1 || results[0].Kind != types.KindClass {
		t.Errorf("expected ParentRecord class, got %+v", results)
	}
}
