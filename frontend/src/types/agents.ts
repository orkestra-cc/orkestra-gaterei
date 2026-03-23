// --- Project Types ---

export interface AgentProject {
  uuid: string;
  name: string;
  description: string;
  hindsightBankId: string;
  documentUuids: string[];
  isoStandards?: string[];
  categories?: string[];
  status: 'active' | 'archived';
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateProjectRequest {
  name: string;
  description: string;
  documentUuids?: string[];
  isoStandards?: string[];
  categories?: string[];
}

export interface UpdateProjectRequest {
  name?: string;
  description?: string;
  status?: 'active' | 'archived';
}

// --- Conversation Types ---

export interface AgentConversation {
  uuid: string;
  projectUuid: string;
  userUuid: string;
  persona: string;
  title?: string;
  messages: AgentMessage[];
  createdAt: string;
  updatedAt: string;
}

export interface AgentMessage {
  role: 'user' | 'assistant';
  content: string;
  sources?: AgentSource[];
  metadata?: AgentMsgMeta;
  createdAt: string;
}

export interface AgentSource {
  documentUuid: string;
  documentTitle: string;
  chunkText: string;
  fullPath: string;
  requirementLevel?: string;
  score: number;
}

export interface AgentMsgMeta {
  ragTimeMs?: number;
  reflectTimeMs?: number;
  totalTimeMs?: number;
  chunksRetrieved?: number;
  modelUsed?: string;
}

// --- Query Types ---

export interface AgentQueryRequest {
  question: string;
  persona?: string;
  conversationId?: string;
  topK?: number;
  minScore?: number;
  retrievalMode?: 'vector' | 'graph' | 'hybrid';
}

export interface AgentQueryResponse {
  answer: string;
  sources: AgentSource[];
  conversationId: string;
  metadata: AgentMsgMeta;
}

// --- Persona Types ---

export type PersonaType = 'developer' | 'administrator' | 'manager' | 'auditor' | 'guest';

export const PERSONA_LABELS: Record<PersonaType, string> = {
  developer: 'Developer',
  administrator: 'Administrator',
  manager: 'Manager',
  auditor: 'Auditor',
  guest: 'Guest',
};

export const PERSONA_DESCRIPTIONS: Record<PersonaType, string> = {
  developer: 'Technical details, raw data, section numbers',
  administrator: 'Comprehensive, compliance + management',
  manager: 'Summaries, business impact, risk',
  auditor: 'Evidence-based, compliance status, citations',
  guest: 'General overviews, no internal details',
};
