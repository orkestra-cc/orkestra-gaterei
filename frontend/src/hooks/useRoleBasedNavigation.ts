/**
 * Custom hook for filtering navigation based on user roles and permissions
 */

import { useMemo } from 'react';
import { useAuth } from './redux/useAuth';
import { NavItem, RouteGroup } from '../routes/siteMaps';
import {
  hasAnyRole,
  extractUserRole,
  UserRole
} from '../utils/roleUtils';

interface UseRoleBasedNavigationOptions {
  /**
   * Whether to show empty groups (groups with no visible children)
   * @default false
   */
  showEmptyGroups?: boolean;

  /**
   * Whether to check permissions in addition to roles
   * @default true
   */
  checkPermissions?: boolean;
}

/**
 * Hook to filter navigation items based on user's role and permissions
 */
export const useRoleBasedNavigation = (
  routeGroups: RouteGroup[],
  options: UseRoleBasedNavigationOptions = {}
) => {
  const {
    showEmptyGroups = false,
    checkPermissions = true
  } = options;

  const { user, isAuthenticated, hasPermission, hasAnyPermission } = useAuth();

  // Memoized filtered navigation based on user role and permissions
  const filteredNavigation = useMemo(() => {
    // If user is not authenticated, return empty navigation
    if (!isAuthenticated || !user) {
      console.log('🔐 Navigation: Not authenticated or no user', { isAuthenticated, user });
      return [];
    }

    const userRole = extractUserRole(user);

    // Debug log in development
    if (process.env.NODE_ENV === 'development') {
      console.log('🔐 Navigation: User role extraction', {
        originalRole: user?.role,
        normalizedRole: userRole,
        isAuthenticated
      });
    }

    // If no valid role found, return empty navigation
    if (!userRole) {
      console.warn('🔐 Navigation: No valid role found, returning empty navigation');
      return [];
    }

    /**
     * Check if user can access a navigation item
     */
    const canAccessNavItem = (navItem: NavItem): boolean => {
      // Check role-based access
      if (navItem.roles && navItem.roles.length > 0) {
        const hasRequiredRole = hasAnyRole(userRole, navItem.roles as UserRole[]);
        if (!hasRequiredRole) {
          return false;
        }
      }

      // Check permission-based access
      if (checkPermissions && navItem.permissions && navItem.permissions.length > 0) {
        const hasRequiredPermissions = navItem.permissions.every(permission =>
          hasPermission(permission)
        );
        if (!hasRequiredPermissions) {
          return false;
        }
      }

      return true;
    };

    /**
     * Recursively filter navigation items
     */
    const filterNavItems = (navItems: NavItem[]): NavItem[] => {
      return navItems
        .filter(canAccessNavItem)
        .map(navItem => {
          // If item has children, recursively filter them
          if (navItem.children && navItem.children.length > 0) {
            const filteredChildren = filterNavItems(navItem.children);

            // Return item with filtered children
            return {
              ...navItem,
              children: filteredChildren
            };
          }

          return navItem;
        })
        .filter(navItem => {
          // Remove items with children if all children were filtered out
          if (navItem.children) {
            return navItem.children.length > 0;
          }
          return true;
        });
    };

    /**
     * Filter route groups
     */
    const filteredGroups = routeGroups
      .map(group => {
        // Check if user can access the entire group
        let canAccessGroup = true;

        if (group.roles && group.roles.length > 0) {
          canAccessGroup = hasAnyRole(userRole, group.roles as UserRole[]);
        }

        if (checkPermissions && canAccessGroup && group.permissions && group.permissions.length > 0) {
          canAccessGroup = group.permissions.every(permission =>
            hasPermission(permission)
          );
        }

        if (!canAccessGroup) {
          return null;
        }

        // Filter children of the group
        const filteredChildren = filterNavItems(group.children);

        return {
          ...group,
          children: filteredChildren
        };
      })
      .filter(group => {
        if (!group) return false;

        // Remove empty groups if specified
        if (!showEmptyGroups && group.children.length === 0) {
          return false;
        }

        return true;
      }) as RouteGroup[];

    // Debug log filtered results
    if (process.env.NODE_ENV === 'development') {
      console.log('🔐 Navigation: Filtered groups', {
        totalGroups: routeGroups.length,
        filteredGroupsCount: filteredGroups.length,
        groupLabels: filteredGroups.map(g => g.label)
      });
    }

    return filteredGroups;
  }, [
    routeGroups,
    user,
    isAuthenticated,
    hasPermission,
    hasAnyPermission,
    showEmptyGroups,
    checkPermissions
  ]);

  return {
    filteredNavigation,
    userRole: extractUserRole(user),
    isAuthenticated
  };
};

/**
 * Hook to check if user can access a specific route
 */
export const useCanAccessRoute = (navItem: NavItem): boolean => {
  const { user, isAuthenticated, hasPermission } = useAuth();

  return useMemo(() => {
    if (!isAuthenticated || !user) {
      return false;
    }

    const userRole = extractUserRole(user);
    if (!userRole) {
      return false;
    }

    // Check role-based access
    if (navItem.roles && navItem.roles.length > 0) {
      const hasRequiredRole = hasAnyRole(userRole, navItem.roles as UserRole[]);
      if (!hasRequiredRole) {
        return false;
      }
    }

    // Check permission-based access
    if (navItem.permissions && navItem.permissions.length > 0) {
      const hasRequiredPermissions = navItem.permissions.every(permission =>
        hasPermission(permission)
      );
      if (!hasRequiredPermissions) {
        return false;
      }
    }

    return true;
  }, [user, isAuthenticated, navItem]);
};

export default useRoleBasedNavigation;