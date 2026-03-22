# RAG Module

Retrieval-Augmented Generation system for ISO norm compliance. Manages AI model configs, document ingestion into the knowledge graph, and question answering with source citations.

## Architecture

```
handlers/
  ├── model_handler.go      ← AI model CRUD + connectivity testing
  ├── document_handler.go   ← Document upload + ingestion
  └── query_handler.go      ← RAG query execution
services/
  ├── model_service.go      ← Model management, provider factory, seeding
  ├── ingestion_service.go  ← Full pipeline: extract → chunk → embed → graph
  ├── query_service.go      ← Vector search → context expansion → LLM
  ├── chunker.go            ← Text chunking with ISO heading detection
  └── text_extractor.go     ← PDF/text extraction
providers/
  ├── embedding.go          ← EmbeddingProvider interface
  ├── llm.go                ← LLMProvider interface
  ├── ollama.go             ← Ollama implementation (REST API)
  ├── openai.go             ← OpenAI + compatible APIs (llama.cpp, vLLM)
  └── factory.go            ← Provider creation + model fetching
repository/
  ├── model_repository.go   ← MongoDB: rag_models collection
  └── document_repository.go ← MongoDB: rag_documents collection
```

## Data Storage Split

| Data | Storage | Why |
|------|---------|-----|
| Model configs | MongoDB `rag_models` | CRUD metadata |
| Document metadata + status | MongoDB `rag_documents` | Status tracking, file info |
| Document graph nodes | Memgraph `:RagDocument` | Graph traversal |
| Chunk nodes + embeddings | Memgraph `:RagChunk` | Vector search + context |
| Relationships | Memgraph `HAS_CHUNK`, `NEXT` | Graph RAG context expansion |

## Endpoints

### Models (`/v1/rag/models`)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/` | Create model config |
| GET | `/` | List models (`?type=embedding\|llm`) |
| GET | `/{uuid}` | Get model |
| PATCH | `/{uuid}` | Update model |
| DELETE | `/{uuid}` | Delete model |
| POST | `/{uuid}/default` | Set as default for its type |
| POST | `/{uuid}/test` | Test provider connectivity |
| POST | `/fetch` | Fetch available models from provider URL |

### Documents (`/v1/rag/documents`)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/` | Upload + ingest (multipart form: file, title, isoStandard, version) |
| GET | `/` | List documents (`?status=...&isoStandard=...`) |
| GET | `/{uuid}` | Get document details |
| DELETE | `/{uuid}` | Delete from MongoDB + Memgraph |

### Query (`/v1/rag/query`)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/` | Ask question, get answer with sources |

## AI Provider System

Two provider interfaces with multiple implementations:

### EmbeddingProvider
```go
type EmbeddingProvider interface {
    Embed(ctx, text) ([]float64, error)
    EmbedBatch(ctx, texts) ([][]float64, error)
    Dimensions() int
    ModelName() string
}
```

### LLMProvider
```go
type LLMProvider interface {
    Complete(ctx, prompt, opts) (string, error)
    StreamComplete(ctx, prompt, opts) (<-chan StreamChunk, error)
    ModelName() string
}
```

### Supported Providers

| Provider | Config | Notes |
|----------|--------|-------|
| `openai` | BaseURL + optional APIKey | Works with OpenAI cloud, llama.cpp, vLLM, LocalAI |
| `ollama` | BaseURL | Ollama REST API (`/api/embeddings`, `/api/generate`) |

The `openai` provider supports custom base URLs via `openai.NewClientWithConfig()`, making it compatible with any OpenAI-compatible API server.

**Current defaults**: llama.cpp at `192.168.88.52` — embedding on `:8082/v1`, LLM on `:8081/v1`.

### API Key Security

- `APIKey` field has `json:"-"` tag — never returned in API responses
- Global fallback: `OPENAI_API_KEY` env var applied when model-level key is empty
- Local servers (llama.cpp, Ollama) don't require API keys

## Document Ingestion Pipeline

Runs asynchronously in a background goroutine (17 steps):

```
1.  Create MongoDB record (status: "pending")
2.  Update status → "processing"
3.  Extract text (TextExtractor: PDF or plain text)
4.  Parse document structure (structural_parser.go: builds hierarchical tree)
5.  Chunk using structural boundaries (chunker.go: ChunkStructured)
6.  Generate embeddings (EmbeddingProvider.Embed per chunk)
7.  Create :RagDocument node in Memgraph
8.  Create :RagSection nodes from structural tree
9.  Create HAS_SECTION relationships (document → top-level sections)
10. Create CONTAINS relationships (section → child sections)
11. Create NEXT_SECTION relationships (sequential siblings)
12. Create :RagChunk nodes with embeddings
13. Create CONTAINS relationships (section → chunks)
14. Create NEXT relationships between sequential chunks
15. Extract definitions → :RagDefinition nodes + DEFINES edges
16. Resolve cross-references → REFERENCES edges
17. Compute SIMILAR_TO edges (cosine > 0.85 threshold)
18. Create vector + property indexes
19. Update status → "completed"
```

### Memgraph Graph Schema

```cypher
(:RagDocument {uuid, title, isoStandard, version, documentCategory, docType, chunkCount, createdAt})
(:RagSection {uuid, documentUuid, nodeType, numbering, title, depth, fullPath, position})
(:RagChunk {uuid, documentUuid, text, position, nodeType, numbering, fullPath, requirementLevel, depth, sectionUuid, embedding})
(:RagDefinition {uuid, documentUuid, term, definition, embedding})

(RagDocument)-[:HAS_SECTION]->(RagSection)
(RagSection)-[:CONTAINS]->(RagSection)
(RagSection)-[:CONTAINS]->(RagChunk)
(RagSection)-[:NEXT_SECTION]->(RagSection)
(RagChunk)-[:NEXT]->(RagChunk)
(RagDocument)-[:HAS_DEFINITION]->(RagDefinition)
(RagDefinition)-[:DEFINES]->(RagChunk)
(RagChunk)-[:REFERENCES {referenceText}]->(RagSection)
(RagChunk)-[:SIMILAR_TO {similarity}]->(RagChunk)
```

### Structural Chunking

The chunker uses a two-phase approach:

**Phase 1 — Structure Detection** (`structural_parser.go`):
- State-machine parser processes text line-by-line with ~15 regex patterns
- Supports ISO standards (Clause, Section, Annex) AND Italian legal docs (TITOLO, CAPO, SEZIONE, Articolo)
- Builds a hierarchical `StructuralNode` tree
- Detects requirement language: SHALL/SHOULD/MAY (EN) + deve/dovrebbe/può (IT)
- Extracts cross-references: "see 4.1.3", "Art. 12", "Annex A", etc.

**Phase 2 — Tree → Chunks** (`chunker.go`):
- Leaf-first: each leaf node becomes a chunk (128–1024 chars)
- Merges small adjacent siblings under same parent
- Splits oversized nodes at sentence boundaries
- Every chunk inherits `fullPath` metadata (e.g., "Clause 4 > 4.1 > 4.1.2")
- No overlap needed — context recovered via graph traversal

### Relationship Extraction (`relationship_extractor.go`)

- **Definitions**: Parsed from "Terms and definitions" sections → `RagDefinition` nodes
- **Cross-references**: Regex-based, resolved to target `RagSection` by numbering
- **Similarity**: Pairwise cosine between non-adjacent chunks, edges for pairs > 0.85

## RAG Query Pipeline

```
1. Embed question → default EmbeddingProvider
2. Vector search with optional filters (requirementLevel, nodeType)
3. Context expansion:
   - "vector" mode: prev/next chunks via NEXT edges
   - "graph" mode: section hierarchy + cross-refs + definitions
4. Fetch document metadata → title, isoStandard from :RagDocument nodes
5. Build richer prompt with structural paths + requirement levels + definitions
6. LLM generation → default LLMProvider.Complete() with 5-minute timeout
7. Return → answer + source citations + timing metadata
```

### System Prompt

The LLM is instructed as an ISO compliance expert. `/no_think` is appended to disable reasoning mode in models like Qwen 3.5.

### Timeouts

- LLM generation uses a dedicated `context.WithTimeout(5 * time.Minute)` — separate from the HTTP request context
- HTTP server WriteTimeout set to 5 minutes to accommodate local LLM inference
- Embedding calls use standard HTTP client timeout (120s)

## Configuration

```
RAG_ENABLED=true              # Enable/disable module
OLLAMA_BASE_URL=...           # Default Ollama endpoint
OPENAI_API_KEY=...            # Default OpenAI API key
RAG_CHUNK_SIZE=512            # Default chunk size (chars)
RAG_CHUNK_OVERLAP=50          # Default overlap (chars)
RAG_DEFAULT_TOP_K=10          # Default vector search results
```

## Default Model Seeding

On first startup (when `rag_models` collection is empty), seeds two models:
- **Embedding**: Qwen3-Embedding-8B at `192.168.88.52:8082/v1` (4096d)
- **LLM**: Qwen3.5-9B at `192.168.88.52:8081/v1`

Both use the `openai` provider (llama.cpp is OpenAI-compatible).

## Adding a New Provider

1. Implement `EmbeddingProvider` and/or `LLMProvider` in `providers/`
2. Add a case to `NewEmbeddingProvider`/`NewLLMProvider` in `factory.go`
3. Add a case to `TestConnectivity` and `FetchAvailableModels` in `factory.go`
4. Update `ModelConfig.Provider` enum in DTOs
