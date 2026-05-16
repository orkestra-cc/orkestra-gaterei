// Observability types — ADR-0005 Phase F.
// Mirrors backend/internal/core/logging/models.

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export const LOG_LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error'];

export interface AdminModuleEntry {
  name: string;
  effective: LogLevel;
  hasOverride: boolean;
}

export interface LogLevelsView {
  global: LogLevel;
  modules: AdminModuleEntry[];
  updatedAt: string;
  updatedBy?: string;
}

export interface SetLevelBody {
  level: LogLevel;
}
