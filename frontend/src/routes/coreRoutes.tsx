import { Suspense, lazy } from 'react';
import { Navigate, type RouteObject } from 'react-router';
import App from 'App';
import paths from './paths';
import MainLayout from 'layouts/MainLayout';
import ErrorLayout from 'layouts/ErrorLayout';
import ProtectedRoute from 'components/authentication/ProtectedRoute';
import FalconLoader from 'components/common/FalconLoader';

import Error401 from 'components/errors/Error401';
import Error404 from 'components/errors/Error404';
import Error500 from 'components/errors/Error500';

import Login from 'components/authentication/Login';
import Register from 'components/authentication/Register';
import ForgotPassword from 'components/authentication/ForgotPassword';
import ResetPassword from 'components/authentication/ResetPassword';
import VerifyEmailPage from 'components/authentication/VerifyEmailPage';
import SocialAuthCallback from 'components/authentication/SocialAuthCallback';

const SetupWizard = lazy(() => import('pages/setup/SetupWizard'));
const UserManagement = lazy(() => import('pages/admin/users'));
const ModuleManagement = lazy(() => import('pages/admin/modules'));
const RoleManagement = lazy(() => import('pages/admin/roles'));
const TenantManagement = lazy(() => import('pages/admin/tenants'));
const AdminUserProfile = lazy(
  () => import('pages/admin/user-profile/AdminUserProfile')
);
const OperatorProfile = lazy(
  () => import('pages/operator/profile/OperatorProfile')
);
const UserDashboard = lazy(
  () => import('pages/user/dashboard/UserDashboard')
);
const UserCalendar = lazy(
  () => import('pages/user/calendar/UserCalendar')
);
const Settings = lazy(() => import('pages/user/settings/Settings'));

/**
 * Builds the full route tree with core routes + injected module/reference children.
 * Module and reference routes are injected as siblings of the admin/user routes
 * inside the protected MainLayout.
 */
export function buildCoreRoutes(
  additionalChildren: RouteObject[]
): RouteObject[] {
  return [
    {
      element: <App />,
      children: [
        {
          path: 'landing',
          element: <Navigate to="/login" replace />,
        },
        {
          path: 'errors',
          element: <ErrorLayout />,
          children: [
            { path: '401', element: <Error401 /> },
            { path: '404', element: <Error404 /> },
            { path: '500', element: <Error500 /> },
          ],
        },
        {
          path: 'setup',
          element: (
            <Suspense key="setup-wizard" fallback={<FalconLoader />}>
              <SetupWizard />
            </Suspense>
          ),
        },
        { path: 'login', element: <Login /> },
        { path: 'register', element: <Register /> },
        { path: 'forgot-password', element: <ForgotPassword /> },
        { path: 'reset-password', element: <ResetPassword /> },
        { path: 'verify-email', element: <VerifyEmailPage /> },
        { path: 'auth/callback', element: <SocialAuthCallback /> },
        {
          path: '/',
          element: (
            <ProtectedRoute>
              <MainLayout />
            </ProtectedRoute>
          ),
          children: [
            {
              index: true,
              element: <Navigate to="/user/dashboard" replace />,
            },
            // ── Admin routes (core) ──
            {
              path: 'admin',
              children: [
                {
                  path: 'users',
                  element: (
                    <ProtectedRoute
                      requiredPermissions={[
                        ['super_admin', 'administrator', 'developer'],
                      ]}
                    >
                      <Suspense
                        key="admin-userManagement"
                        fallback={<FalconLoader />}
                      >
                        <UserManagement />
                      </Suspense>
                    </ProtectedRoute>
                  ),
                },
                {
                  path: 'modules',
                  element: (
                    <ProtectedRoute
                      requiredPermissions={[
                        ['super_admin', 'administrator', 'developer'],
                      ]}
                    >
                      <Suspense
                        key="admin-modules"
                        fallback={<FalconLoader />}
                      >
                        <ModuleManagement />
                      </Suspense>
                    </ProtectedRoute>
                  ),
                },
                {
                  path: 'roles',
                  element: (
                    <ProtectedRoute
                      requiredPermissions={[
                        [
                          'authz.role.read',
                          'super_admin',
                          'administrator',
                          'developer',
                        ],
                      ]}
                    >
                      <Suspense
                        key="admin-roles"
                        fallback={<FalconLoader />}
                      >
                        <RoleManagement />
                      </Suspense>
                    </ProtectedRoute>
                  ),
                },
                {
                  path: 'tenants',
                  element: (
                    <ProtectedRoute
                      requiredPermissions={[
                        ['super_admin', 'administrator', 'developer'],
                      ]}
                    >
                      <Suspense
                        key="admin-tenants"
                        fallback={<FalconLoader />}
                      >
                        <TenantManagement />
                      </Suspense>
                    </ProtectedRoute>
                  ),
                },
                {
                  path: 'user/profile/:userId',
                  element: (
                    <ProtectedRoute
                      requiredPermissions={[
                        ['super_admin', 'administrator', 'developer'],
                      ]}
                    >
                      <Suspense
                        key="admin-userProfile"
                        fallback={<FalconLoader />}
                      >
                        <AdminUserProfile />
                      </Suspense>
                    </ProtectedRoute>
                  ),
                },
              ],
            },
            // ── User / Operator routes (core) ──
            {
              path: 'user',
              children: [
                {
                  path: 'profile',
                  element: (
                    <Suspense
                      key="operator-profile"
                      fallback={<FalconLoader />}
                    >
                      <OperatorProfile />
                    </Suspense>
                  ),
                },
                {
                  path: 'dashboard',
                  element: (
                    <Suspense
                      key="user-dashboard"
                      fallback={<FalconLoader />}
                    >
                      <UserDashboard />
                    </Suspense>
                  ),
                },
                {
                  path: 'calendar',
                  element: (
                    <Suspense
                      key="user-calendar"
                      fallback={<FalconLoader />}
                    >
                      <UserCalendar />
                    </Suspense>
                  ),
                },
                {
                  path: 'settings',
                  element: (
                    <Suspense
                      key="user-settings"
                      fallback={<FalconLoader />}
                    >
                      <Settings />
                    </Suspense>
                  ),
                },
              ],
            },
            // ── Module + Reference routes (injected) ──
            ...additionalChildren,
          ],
        },
        {
          path: '*',
          element: <Navigate to={paths.error404} replace />,
        },
      ],
    },
  ];
}
