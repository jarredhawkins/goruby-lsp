package index

import "github.com/jarredhawkins/goruby-lsp/internal/types"

// Re-export types for backwards compatibility
type Symbol = types.Symbol
type SymbolKind = types.SymbolKind
type Reference = types.Reference

// Re-export constants
const (
	KindClass           = types.KindClass
	KindModule          = types.KindModule
	KindMethod          = types.KindMethod
	KindSingletonMethod = types.KindSingletonMethod
	KindConstant        = types.KindConstant
	KindAttrReader      = types.KindAttrReader
	KindAttrWriter      = types.KindAttrWriter
	KindAttrAccessor    = types.KindAttrAccessor
	KindLocalVariable   = types.KindLocalVariable
	KindCustom          = types.KindCustom
)
