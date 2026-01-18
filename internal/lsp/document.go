package lsp

import (
	"sync"
)

// DocumentStore manages open text documents
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// Document represents an open text document
type Document struct {
	URI     string
	Version int
	Content string
}

// NewDocumentStore creates a new document store
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		docs: make(map[string]*Document),
	}
}

// Open adds or updates a document
func (ds *DocumentStore) Open(uri string, version int, content string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.docs[uri] = &Document{
		URI:     uri,
		Version: version,
		Content: content,
	}
}

// Update updates a document's content
func (ds *DocumentStore) Update(uri string, version int, content string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if doc, ok := ds.docs[uri]; ok {
		doc.Version = version
		doc.Content = content
	}
}

// Close removes a document
func (ds *DocumentStore) Close(uri string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.docs, uri)
}

// Get returns a document's content
func (ds *DocumentStore) Get(uri string) (string, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	if doc, ok := ds.docs[uri]; ok {
		return doc.Content, true
	}
	return "", false
}

// IsOpen checks if a document is open
func (ds *DocumentStore) IsOpen(uri string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	_, ok := ds.docs[uri]
	return ok
}
