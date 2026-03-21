// --- Model Types ---

export interface ModelConfig {
  uuid: string;
  name: string;
  provider: 'ollama' | 'openai';
  modelType: 'embedding' | 'llm';
  modelName: string;
  baseUrl?: string;
  dimensions?: number;
  temperature?: number;
  maxTokens?: number;
  isDefault: boolean;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateModelRequest {
  name: string;
  provider: 'ollama' | 'openai';
  modelType: 'embedding' | 'llm';
  modelName: string;
  baseUrl?: string;
  apiKey?: string;
  dimensions?: number;
  temperature?: number;
  maxTokens?: number;
}

export interface UpdateModelRequest {
  name?: string;
  baseUrl?: string;
  apiKey?: string;
  dimensions?: number;
  temperature?: number;
  maxTokens?: number;
  isActive?: boolean;
}

export interface TestModelResult {
  status: 'ok' | 'error';
  message: string;
}

// --- Document Types ---

export interface RagDocument {
  uuid: string;
  title: string;
  fileName: string;
  fileSize: number;
  isoStandard?: string;
  version?: string;
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

// --- Query Types ---

export interface RagQueryRequest {
  question: string;
  topK?: number;
  minScore?: number;
  isoStandard?: string;
  modelUuid?: string;
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
  sectionTitle?: string;
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
