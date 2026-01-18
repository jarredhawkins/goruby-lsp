package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jarredhawkins/goruby-lsp/internal/index"
	"github.com/jarredhawkins/goruby-lsp/internal/lsp"
	"github.com/jarredhawkins/goruby-lsp/internal/parser"
	"github.com/jarredhawkins/goruby-lsp/internal/watcher"
)

func main() {
	var (
		rootPath string
		logFile  string
		debug    bool
	)

	flag.StringVar(&rootPath, "root", "", "Root path of the Ruby project (defaults to current directory)")
	flag.StringVar(&logFile, "log", "", "Log file path (defaults to stderr)")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	// Default to current directory
	if rootPath == "" {
		var err error
		rootPath, err = os.Getwd()
		if err != nil {
			log.Fatalf("failed to get current directory: %v", err)
		}
	}

	// Setup logging
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	if debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	log.Printf("ruby-lsp starting, root=%s", rootPath)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown signal received")
		cancel()
	}()

	// Initialize parser registry with default matchers
	registry := parser.NewRegistry()
	parser.RegisterDefaults(registry)

	// Create and build the index
	idx := index.New(rootPath, registry)
	if err := idx.Build(ctx); err != nil {
		log.Fatalf("failed to build index: %v", err)
	}

	// Start file watcher
	w, err := watcher.New(rootPath, func(changed, removed []string) {
		for _, path := range removed {
			idx.RemoveFile(path)
		}
		for _, path := range changed {
			if err := idx.UpdateFile(path); err != nil {
				log.Printf("failed to update file %s: %v", path, err)
			}
		}
	})
	if err != nil {
		log.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		log.Fatalf("failed to start watcher: %v", err)
	}

	// Start LSP server on stdio
	server := lsp.NewServer(idx)
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("LSP server error: %v", err)
	}

	log.Println("ruby-lsp shutdown complete")
}
