package lsp

import (
	"os"
	"strings"

	"github.com/jarredhawkins/goruby-lsp/internal/index"
)

// LSP Protocol types - minimal set for definition and references

// TextDocumentSyncKind defines how text document changes are synced
type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone        TextDocumentSyncKind = 0
	TextDocumentSyncKindFull        TextDocumentSyncKind = 1
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

// Position in a text document
type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

// Range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a resource
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a versioned text document
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem represents an open text document
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentPositionParams is a parameter for requests that require a position
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// ReferenceContext includes info about reference requests
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ReferenceParams for textDocument/references
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// TextDocumentSyncOptions defines text document sync options
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose,omitempty"`
	Change    TextDocumentSyncKind `json:"change,omitempty"`
}

// ServerCapabilities defines what the server can do
type ServerCapabilities struct {
	TextDocumentSync   *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	DefinitionProvider bool                     `json:"definitionProvider,omitempty"`
	ReferencesProvider bool                     `json:"referencesProvider,omitempty"`
}

// ServerInfo contains information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// InitializeResult is the result of the initialize request
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// DidOpenTextDocumentParams for textDocument/didOpen
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentContentChangeEvent describes changes to a text document
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// DidChangeTextDocumentParams for textDocument/didChange
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams for textDocument/didClose
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Helper functions

// uriToPath converts a file:// URI to a file path
func uriToPath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}

// pathToURI converts a file path to a file:// URI
func pathToURI(path string) string {
	if strings.HasPrefix(path, "file://") {
		return path
	}
	return "file://" + path
}

// symbolToLocation converts an index.Symbol to an LSP Location
func symbolToLocation(sym *index.Symbol) Location {
	return Location{
		URI: pathToURI(sym.FilePath),
		Range: Range{
			Start: Position{
				Line:      uint32(sym.Line - 1), // LSP is 0-indexed
				Character: uint32(sym.Column),
			},
			End: Position{
				Line:      uint32(sym.Line - 1),
				Character: uint32(sym.Column + len(sym.Name)),
			},
		},
	}
}

// extractWordAt extracts the word at the given position in the content
func extractWordAt(content string, line, char int) string {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}

	lineText := lines[line]
	if char < 0 || char >= len(lineText) {
		// Try to find the last word if char is at/past end
		if char >= len(lineText) && len(lineText) > 0 {
			char = len(lineText) - 1
		} else {
			return ""
		}
	}

	// If cursor is on a Ruby method suffix (? ! =), move back into the word
	if char < len(lineText) {
		ch := lineText[char]
		if ch == '?' || ch == '!' || ch == '=' {
			if char > 0 && isWordChar(lineText[char-1]) {
				char-- // Move into the word part
			}
		}
	}

	// Find word boundaries
	// Ruby identifiers: letters, digits, underscores, and can end with ? ! =
	start := char
	for start > 0 && isWordChar(lineText[start-1]) {
		start--
	}

	end := char
	for end < len(lineText) && isWordChar(lineText[end]) {
		end++
	}

	// Include trailing ? ! = for Ruby methods
	if end < len(lineText) {
		ch := lineText[end]
		if ch == '?' || ch == '!' || ch == '=' {
			end++
		}
	}

	if start == end {
		return ""
	}

	return lineText[start:end]
}

// isWordChar returns true if c is a valid Ruby identifier character
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// readFile reads a file from disk
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
