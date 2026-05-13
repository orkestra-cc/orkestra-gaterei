// Package config carries the typed `RAGConfig` DTO that the rag
// addon's service constructors accept. It used to live at
// backend/internal/shared/config (alongside every other module's
// config struct) and was relocated here as part of Phase 5l of the
// SDK split, when the addon was carved into its own Go module.
//
// The shared config package's env-var populator was already dead
// code by the time the relocation happened: `module.go` builds a
// fresh RAGConfig value from its own `Settings` struct (unmarshaled
// via the SDK's ConfigService from the `module_configs` collection),
// so nothing reads `config.RAG` from the kernel-side struct anymore.
package config

// RAGConfig holds runtime configuration for the RAG addon. Built once
// in module.Init() from the live ConfigService snapshot and threaded
// into the model service. Enabled is always true at the construction
// site — the registry has already gated activation on the module's
// configured enabled state before Init runs.
type RAGConfig struct {
	Enabled       bool   // Module enabled flag (RAG_ENABLED) — true by construction inside this addon's Init
	OllamaBaseURL string // Ollama API base URL
	OpenAIAPIKey  string // OpenAI API key
	ChunkSize     int    // Default text chunk size in characters
	ChunkOverlap  int    // Overlap between chunks in characters
	DefaultTopK   int    // Default number of results for vector search
}
