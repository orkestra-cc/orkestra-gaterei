package providers

import (
	"context"

	"github.com/orkestra/backend/internal/shared/iface"
)

// EmbeddingProvider, LLMProvider, CompletionOptions, and StreamChunk used
// to be defined here. They now live in shared/iface/aimodels_types.go so
// the iface contract layer doesn't import this addon package. These thin
// type aliases let provider implementations and existing cross-module
// callers (sales, rag) keep using the `providers.LLMProvider` form
// unchanged — the underlying type is identical.
type (
	EmbeddingProvider = iface.EmbeddingProvider
	LLMProvider       = iface.LLMProvider
	CompletionOptions = iface.CompletionOptions
	StreamChunk       = iface.StreamChunk
)

// CompletionResult holds the LLM response text along with token usage information.
// Stays in this addon package — only sales (a cross-module caller via type
// assertion to LLMProviderWithUsage) sees it, no need to promote to iface.
type CompletionResult struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

// LLMProviderWithUsage extends LLMProvider with token usage tracking.
// Providers that support usage reporting (e.g. Anthropic) implement this.
type LLMProviderWithUsage interface {
	LLMProvider
	CompleteWithUsage(ctx context.Context, prompt string, opts CompletionOptions) (*CompletionResult, error)
}
