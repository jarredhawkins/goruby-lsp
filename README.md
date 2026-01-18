# goruby-lsp

A fast, lightweight LSP server for Ruby projects written in Go.

Provides **go to definition** and **find references** for Ruby codebases with minimal setup and near-instant startup.

## Installation

```bash
go install github.com/jarredhawkins/goruby-lsp/cmd/goruby-lsp@latest
```

Or build from source:

```bash
git clone https://github.com/jarredhawkins/goruby-lsp
cd goruby-lsp
go install ./cmd/goruby-lsp
```

## Usage

Run from your Ruby project root:

```bash
cd /path/to/your/rails/app
goruby-lsp
```

The server communicates over stdio using the Language Server Protocol.

### CLI Flags

| Flag | Description |
|------|-------------|
| `--root <path>` | Root path of the Ruby project (defaults to cwd) |
| `--log <file>` | Log file path (defaults to stderr) |
| `--debug` | Enable debug logging |

### Editor Setup

**VS Code**: Add to `.vscode/settings.json`:
```json
{
  "ruby.languageServer": {
    "command": "goruby-lsp"
  }
}
```

**Neovim** (with nvim-lspconfig):
```lua
require('lspconfig.configs').goruby_lsp = {
  default_config = {
    cmd = { 'goruby-lsp' },
    filetypes = { 'ruby' },
    root_dir = require('lspconfig.util').root_pattern('Gemfile', '.git'),
  },
}
require('lspconfig').goruby_lsp.setup({})
```

## Features

- **textDocument/definition** - Jump to class, module, method, and constant definitions
- **textDocument/references** - Find all usages of a symbol using trigram search
- **Live reindexing** - File changes are detected via fsnotify and the index updates automatically

## Architecture

### How It Works

1. On startup, walks the project tree and indexes all `.rb` files
2. Parses Ruby files line-by-line using regex patterns to extract definitions
3. Builds an in-memory symbol index and trigram index for fast lookups
4. Watches for file changes and incrementally updates the index

### Supported Ruby Constructs

| Construct | Example |
|-----------|---------|
| Classes | `class MyClass`, `class MyModule::MyClass < Base` |
| Modules | `module MyModule` |
| Methods | `def my_method`, `def self.class_method` |
| Constants | `MY_CONST = value` |

The parser uses a plugin system—additional patterns (like `attr_accessor`, Rails DSLs) can be added.

## Tradeoffs

This project uses **regex-based parsing** rather than a proper AST parser like [tree-sitter](https://tree-sitter.github.io/tree-sitter/). Here's why, and what you give up:

### Why Regex?

| Benefit | Explanation |
|---------|-------------|
| **No CGO** | Tree-sitter requires C bindings, complicating cross-compilation and builds |
| **Fast startup** | No grammar loading or parser initialization overhead |
| **Simple plugins** | Adding new patterns is just writing a regex |
| **Small binary** | ~5MB vs 15MB+ with tree-sitter grammars |

### What You Lose

| Limitation | Impact |
|------------|--------|
| **No AST** | Can't resolve scope accurately in complex cases |
| **Edge cases** | Misses definitions inside heredocs, multiline strings, or unusual formatting |
| **No type inference** | Can't follow `include`/`extend` to find inherited methods |
| **Metaprogramming** | `define_method`, `class_eval`, etc. are invisible |

### When to Use This vs. Solargraph/Ruby LSP

| Use goruby-lsp when... | Use Solargraph/ruby-lsp when... |
|------------------------|--------------------------------|
| You want instant startup | You need accurate type inference |
| You're on a large monorepo | You need metaprogramming support |
| You just need "good enough" navigation | You need completion, hover, diagnostics |
| You want a single binary with no Ruby deps | You don't mind Ruby/gem dependencies |

## Extending

### Adding a Custom Matcher

Create a new matcher in `internal/plugins/`:

```go
package plugins

import (
    "regexp"
    "github.com/jarredhawkins/goruby-lsp/internal/parser"
    "github.com/jarredhawkins/goruby-lsp/internal/types"
)

var attrPattern = regexp.MustCompile(`^\s*attr_(reader|writer|accessor)\s+(.+)`)

type AttrMatcher struct{}

func (m *AttrMatcher) Name() string  { return "attr" }
func (m *AttrMatcher) Priority() int { return 85 }

func (m *AttrMatcher) Match(line string, ctx *parser.ParseContext) *parser.MatchResult {
    match := attrPattern.FindStringSubmatch(line)
    if match == nil {
        return nil
    }
    // Extract symbols from match...
    return &parser.MatchResult{Symbols: symbols}
}
```

Then register it in `parser.RegisterDefaults()`.

## License

[Unlicense](LICENSE) - Public domain. Do whatever you want.
