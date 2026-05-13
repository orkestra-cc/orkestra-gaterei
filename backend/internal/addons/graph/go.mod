// Orkestra Graph addon — Memgraph-backed knowledge graph with
// Bolt-protocol query execution, schema introspection, MAGE graph
// algorithms (PageRank, Louvain, etc.), and vector similarity search.
// Owns the orkestra-memgraph container via the SDK's InfraContainers
// hook so toggling the module at /admin/modules brings the database
// up or down within the same request. Hosted in-tree at
// backend/internal/addons/graph/ as its own Go module so the same
// source can be consumed by the orkestra monolith AND extracted to a
// standalone repository (orkestra-cc/orkestra-addon-graph) at the
// same import path. The in-tree path is intentionally still under
// backend/internal/ — Go's internal-package rule operates on import
// paths, not filesystem locations, and the import path here does NOT
// contain "internal".

module github.com/orkestra-cc/orkestra-addon-graph

go 1.25.10

require (
	github.com/danielgtaylor/huma/v2 v2.34.1
	github.com/go-chi/chi/v5 v5.2.5
	github.com/neo4j/neo4j-go-driver/v5 v5.28.4
	github.com/orkestra-cc/orkestra-sdk v0.2.0
)
