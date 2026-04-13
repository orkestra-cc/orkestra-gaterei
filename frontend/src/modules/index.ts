import type { ModuleManifest } from './types';
import { billingManifest } from './billing';
import { companyManifest } from './company';
import { graphManifest } from './graph';
import { aimodelsManifest } from './aimodels';
import { ragManifest } from './rag';
import { agentsManifest } from './agents';
import { salesManifest } from './sales';

/** All optional module manifests, keyed by backend module name */
export const moduleCatalog: Record<string, ModuleManifest> = {
  billing: billingManifest,
  company: companyManifest,
  graph: graphManifest,
  aimodels: aimodelsManifest,
  rag: ragManifest,
  agents: agentsManifest,
  sales: salesManifest,
};

export type { ModuleManifest } from './types';
