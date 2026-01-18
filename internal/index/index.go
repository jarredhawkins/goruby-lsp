package index

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jarredhawkins/goruby-lsp/internal/parser"
	"github.com/jarredhawkins/goruby-lsp/internal/types"
)

// Index provides symbol lookup and text search
type Index struct {
	mu sync.RWMutex

	// Primary index: FullName -> definitions
	symbols map[string][]*Symbol

	// Short name index: Name -> FullNames (for fuzzy lookup)
	shortNames map[string][]string

	// File index: FilePath -> symbols in file
	byFile map[string][]*Symbol

	// Trigram index for text search
	trigram *TrigramIndex

	rootPath string
	scanner  *parser.Scanner
}

// New creates a new index for the given root path
func New(rootPath string, registry *parser.Registry) *Index {
	return &Index{
		symbols:    make(map[string][]*Symbol),
		shortNames: make(map[string][]string),
		byFile:     make(map[string][]*Symbol),
		trigram:    NewTrigramIndex(),
		rootPath:   rootPath,
		scanner:    parser.NewScanner(registry),
	}
}

// Build performs the initial indexing of all Ruby files
func (idx *Index) Build(ctx context.Context) error {
	log.Printf("building index for %s", idx.rootPath)

	var files []string
	err := filepath.WalkDir(idx.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip hidden directories and vendor
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only index Ruby files
		if isRubyFile(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("found %d Ruby files", len(files))

	// Index files concurrently
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // Limit concurrency

	for _, file := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := idx.AddFile(path); err != nil {
				log.Printf("failed to index %s: %v", path, err)
			}
		}(file)
	}

	wg.Wait()
	log.Printf("indexed %d symbols", idx.SymbolCount())
	return nil
}

// AddFile parses and indexes a single file
func (idx *Index) AddFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	symbols := idx.scanner.Parse(path, content)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Store in file index
	idx.byFile[path] = symbols

	// Store in symbol indexes
	for _, sym := range symbols {
		// Primary index by full name
		idx.symbols[sym.FullName] = append(idx.symbols[sym.FullName], sym)

		// Short name index
		if !contains(idx.shortNames[sym.Name], sym.FullName) {
			idx.shortNames[sym.Name] = append(idx.shortNames[sym.Name], sym.FullName)
		}
	}

	// Add to trigram index
	idx.trigram.AddFile(path, content)

	return nil
}

// RemoveFile removes all symbols from a file
func (idx *Index) RemoveFile(path string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	symbols := idx.byFile[path]
	delete(idx.byFile, path)

	for _, sym := range symbols {
		// Remove from primary index
		existing := idx.symbols[sym.FullName]
		filtered := make([]*Symbol, 0, len(existing))
		for _, s := range existing {
			if s.FilePath != path {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			delete(idx.symbols, sym.FullName)
		} else {
			idx.symbols[sym.FullName] = filtered
		}

		// Clean up short name index
		fullNames := idx.shortNames[sym.Name]
		if len(idx.symbols[sym.FullName]) == 0 {
			filtered := make([]string, 0, len(fullNames))
			for _, fn := range fullNames {
				if fn != sym.FullName {
					filtered = append(filtered, fn)
				}
			}
			if len(filtered) == 0 {
				delete(idx.shortNames, sym.Name)
			} else {
				idx.shortNames[sym.Name] = filtered
			}
		}
	}

	// Remove from trigram index
	idx.trigram.RemoveFile(path)
}

// UpdateFile removes then re-adds a file
func (idx *Index) UpdateFile(path string) error {
	idx.RemoveFile(path)
	return idx.AddFile(path)
}

// FindDefinitions returns definitions matching the symbol name
// Supports both short names ("MyClass") and full names ("MyModule::MyClass")
func (idx *Index) FindDefinitions(name string) []*Symbol {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Try exact full name match first
	if syms, ok := idx.symbols[name]; ok {
		result := make([]*Symbol, len(syms))
		copy(result, syms)
		return result
	}

	// Try short name lookup
	fullNames, ok := idx.shortNames[name]
	if !ok {
		return nil
	}

	var result []*Symbol
	for _, fullName := range fullNames {
		if syms, ok := idx.symbols[fullName]; ok {
			result = append(result, syms...)
		}
	}
	return result
}

// FindReferences finds all references to the given name using trigram search
func (idx *Index) FindReferences(name string) []*Reference {
	return idx.trigram.Search(name)
}

// FindDefinitionsInFile returns definitions matching the name, preferring those in the given file
func (idx *Index) FindDefinitionsInFile(name, filePath string) []*Symbol {
	all := idx.FindDefinitions(name)
	if len(all) == 0 {
		return nil
	}

	// Sort: same file first
	var sameFile, otherFiles []*Symbol
	for _, sym := range all {
		if sym.FilePath == filePath {
			sameFile = append(sameFile, sym)
		} else {
			otherFiles = append(otherFiles, sym)
		}
	}

	return append(sameFile, otherFiles...)
}

// FindLocalVariable finds a local variable definition in the method containing cursorLine.
// Returns the first matching local variable, or nil if not found.
func (idx *Index) FindLocalVariable(name, filePath string, cursorLine int) *Symbol {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	syms := idx.byFile[filePath]
	if syms == nil {
		return nil
	}

	// Find the method containing cursorLine
	var containingMethod *Symbol
	for _, sym := range syms {
		if (sym.Kind == types.KindMethod || sym.Kind == types.KindSingletonMethod) &&
			sym.Line <= cursorLine && sym.EndLine >= cursorLine {
			containingMethod = sym
			break
		}
	}

	if containingMethod == nil {
		return nil
	}

	// Find first local variable with matching name in that method
	for _, sym := range syms {
		if sym.Kind == types.KindLocalVariable &&
			sym.Name == name &&
			sym.MethodFullName == containingMethod.FullName &&
			sym.Line > containingMethod.Line &&
			sym.Line <= containingMethod.EndLine {
			return sym
		}
	}

	return nil
}

// SymbolsInFile returns all symbols defined in a file
func (idx *Index) SymbolsInFile(path string) []*Symbol {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	syms := idx.byFile[path]
	result := make([]*Symbol, len(syms))
	copy(result, syms)
	return result
}

// SymbolCount returns the total number of indexed symbols
func (idx *Index) SymbolCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	count := 0
	for _, syms := range idx.symbols {
		count += len(syms)
	}
	return count
}

// RootPath returns the root path of the index
func (idx *Index) RootPath() string {
	return idx.rootPath
}

// isRubyFile checks if a file is a Ruby file
func isRubyFile(path string) bool {
	ext := filepath.Ext(path)
	base := filepath.Base(path)

	switch ext {
	case ".rb", ".rake", ".gemspec":
		return true
	}

	switch base {
	case "Gemfile", "Rakefile", "Guardfile", "Vagrantfile":
		return true
	}

	return false
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
