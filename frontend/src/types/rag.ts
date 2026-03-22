// --- Document Types ---

export interface RagDocument {
  uuid: string;
  title: string;
  fileName: string;
  fileSize: number;
  isoStandard?: string;
  version?: string;
  documentCategory?: string;
  docType: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  error?: string;
  chunkCount: number;
  modelUuid: string;
  chunkSize: number;
  chunkOverlap: number;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

// --- Chunk Types ---

export interface RagChunk {
  uuid: string;
  documentUuid: string;
  text: string;
  position: number;
  fullPath?: string;
  nodeType?: string;
  numbering?: string;
  requirementLevel?: string;
  depth?: number;
}

// --- Section Types ---

export interface RagSection {
  uuid: string;
  documentUuid: string;
  nodeType: string;
  numbering?: string;
  title?: string;
  depth: number;
  fullPath?: string;
  position: number;
}

// --- Update Types ---

export interface UpdateDocumentRequest {
  title?: string;
  isoStandard?: string;
  version?: string;
}

// --- Query Types ---

export interface RagQueryRequest {
  question: string;
  topK?: number;
  minScore?: number;
  isoStandard?: string;
  modelUuid?: string;
  requirementLevel?: string;
  nodeType?: string;
  retrievalMode?: string;
}

export interface RagQueryResponse {
  answer: string;
  sources: SourceRef[];
  metadata: QueryMeta;
}

export interface SourceRef {
  documentUuid: string;
  documentTitle: string;
  isoStandard?: string;
  chunkUuid: string;
  chunkText: string;
  fullPath?: string;
  nodeType?: string;
  requirementLevel?: string;
  score: number;
  position: number;
}

export interface QueryMeta {
  embeddingTimeMs: number;
  searchTimeMs: number;
  llmTimeMs: number;
  totalTimeMs: number;
  chunksRetrieved: number;
  modelUsed: string;
}
