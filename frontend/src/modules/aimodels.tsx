import { Suspense, lazy } from 'react';
import type { ModuleManifest } from './types';
import ProtectedRoute from 'components/authentication/ProtectedRoute';
import ModuleGate from 'components/common/ModuleGate';
import FalconLoader from 'components/common/FalconLoader';

const AIModels = lazy(() => import('pages/ai/models'));

export const aimodelsManifest: ModuleManifest = {
  name: 'aimodels',
  routes: () => [
    {
      path: 'ai/models',
      element: (
        <ModuleGate module="aimodels">
          <ProtectedRoute requiredPermissions={[['super_admin', 'administrator', 'developer']]}>
            <Suspense key="ai-models" fallback={<FalconLoader />}>
              <AIModels />
            </Suspense>
          </ProtectedRoute>
        </ModuleGate>
      ),
    },
  ],
  injectApi: () => import('store/api/aiModelsApi'),
};
