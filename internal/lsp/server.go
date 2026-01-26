package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/jarredhawkins/goruby-lsp/internal/index"
	"go.lsp.dev/jsonrpc2"
)

// Server implements the LSP server
type Server struct {
	index     *index.Index
	documents map[string]string // URI -> content cache for open documents
}

// NewServer creates a new LSP server
func NewServer(idx *index.Index) *Server {
	return &Server{
		index:     idx,
		documents: make(map[string]string),
	}
}

// Serve starts the LSP server on the given reader/writer
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	stream := jsonrpc2.NewStream(&readWriteCloser{in, out})
	conn := jsonrpc2.NewConn(stream)

	conn.Go(ctx, s.handler)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.Done():
		return conn.Err()
	}
}

func (s *Server) handler(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	log.Printf("LSP request: %s", req.Method())

	switch req.Method() {
	case "initialize":
		return s.handleInitialize(ctx, reply, req)
	case "initialized":
		return reply(ctx, nil, nil)
	case "shutdown":
		return reply(ctx, nil, nil)
	case "exit":
		return nil
	case "textDocument/definition":
		return s.handleDefinition(ctx, reply, req)
	case "textDocument/references":
		return s.handleReferences(ctx, reply, req)
	case "textDocument/didOpen":
		return s.handleDidOpen(ctx, reply, req)
	case "textDocument/didChange":
		return s.handleDidChange(ctx, reply, req)
	case "textDocument/didClose":
		return s.handleDidClose(ctx, reply, req)
	default:
		// Method not found
		return reply(ctx, nil, &jsonrpc2.Error{
			Code:    jsonrpc2.MethodNotFound,
			Message: "method not supported: " + req.Method(),
		})
	}
}

func (s *Server) handleInitialize(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    TextDocumentSyncKindFull,
			},
			DefinitionProvider: true,
			ReferencesProvider: true,
		},
		ServerInfo: &ServerInfo{
			Name:    "ruby-lsp",
			Version: "0.1.0",
		},
	}
	return reply(ctx, result, nil)
}

func (s *Server) handleDefinition(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, &jsonrpc2.Error{
			Code:    jsonrpc2.InvalidParams,
			Message: err.Error(),
		})
	}

	uri := params.TextDocument.URI
	filePath := uriToPath(uri)
	line := int(params.Position.Line)
	char := int(params.Position.Character)

	// Get document content
	content := s.getDocumentContent(uri)
	if content == "" {
		return reply(ctx, nil, nil)
	}

	// Extract word at position
	word := extractWordAt(content, line, char)
	if word == "" {
		return reply(ctx, nil, nil)
	}

	log.Printf("definition request for word: %s at %s:%d:%d", word, filePath, line, char)

	// Try local variable lookup first (lowercase names only)
	if len(word) > 0 && ((word[0] >= 'a' && word[0] <= 'z') || word[0] == '_') {
		// line is 0-indexed from LSP, FindLocalVariable expects 1-indexed
		if sym := s.index.FindLocalVariable(word, filePath, line+1); sym != nil {
			return reply(ctx, symbolToLocation(sym), nil)
		}
	}

	// Look up definitions in global index
	symbols := s.index.FindDefinitionsInFile(word, filePath)
	if len(symbols) == 0 {
		return reply(ctx, nil, nil)
	}

	// Convert to LSP locations
	if len(symbols) == 1 {
		return reply(ctx, symbolToLocation(symbols[0]), nil)
	}

	locations := make([]Location, len(symbols))
	for i, sym := range symbols {
		locations[i] = symbolToLocation(sym)
	}
	return reply(ctx, locations, nil)
}

func (s *Server) handleReferences(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params ReferenceParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, &jsonrpc2.Error{
			Code:    jsonrpc2.InvalidParams,
			Message: err.Error(),
		})
	}

	uri := params.TextDocument.URI
	line := int(params.Position.Line)
	char := int(params.Position.Character)

	content := s.getDocumentContent(uri)
	if content == "" {
		return reply(ctx, nil, nil)
	}

	word := extractWordAt(content, line, char)
	if word == "" {
		return reply(ctx, nil, nil)
	}

	log.Printf("references request for word: %s", word)

	// Use a map to deduplicate by location key (file:line:col)
	seen := make(map[string]struct{})
	var locations []Location

	// Find all references using trigram search
	refs := s.index.FindReferences(word)
	for _, ref := range refs {
		key := fmt.Sprintf("%s:%d:%d", ref.FilePath, ref.Line, ref.Column)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		locations = append(locations, Location{
			URI: pathToURI(ref.FilePath),
			Range: Range{
				Start: Position{
					Line:      uint32(ref.Line - 1),
					Character: uint32(ref.Column),
				},
				End: Position{
					Line:      uint32(ref.Line - 1),
					Character: uint32(ref.Column + ref.Length),
				},
			},
		})
	}

	// Find symbols that target this name (e.g., relations targeting a class)
	targetingRefs := s.index.FindTargetingSymbols(word)
	for _, sym := range targetingRefs {
		key := fmt.Sprintf("%s:%d:%d", sym.FilePath, sym.Line, sym.Column)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		locations = append(locations, symbolToLocation(sym))
	}

	// Include declarations if requested - deduplication prevents double-adding
	if params.Context.IncludeDeclaration {
		symbols := s.index.FindDefinitions(word)
		for _, sym := range symbols {
			key := fmt.Sprintf("%s:%d:%d", sym.FilePath, sym.Line, sym.Column)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			locations = append(locations, symbolToLocation(sym))
		}
	}

	return reply(ctx, locations, nil)
}

func (s *Server) handleDidOpen(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	s.documents[params.TextDocument.URI] = params.TextDocument.Text
	return reply(ctx, nil, nil)
}

func (s *Server) handleDidChange(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	if len(params.ContentChanges) > 0 {
		// Full sync mode - just take the last content
		s.documents[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
	}
	return reply(ctx, nil, nil)
}

func (s *Server) handleDidClose(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	delete(s.documents, params.TextDocument.URI)
	return reply(ctx, nil, nil)
}

func (s *Server) getDocumentContent(uri string) string {
	// Check open documents first
	if content, ok := s.documents[uri]; ok {
		return content
	}

	// Fall back to reading from disk
	path := uriToPath(uri)
	content, err := readFile(path)
	if err != nil {
		log.Printf("failed to read file %s: %v", path, err)
		return ""
	}
	return content
}

// readWriteCloser wraps reader and writer into a ReadWriteCloser
type readWriteCloser struct {
	io.Reader
	io.Writer
}

func (rwc *readWriteCloser) Close() error {
	return nil
}
