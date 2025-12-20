/**
 * Test utilities for role-based navigation filtering
 * Use these in development to verify role-based access is working correctly
 */

import routeGroups, { NavItem } from '../routes/siteMaps';
import { hasRoleAccess, hasAnyRole, UserRole } from './roleUtils';

interface NavigationTestResult {
  role: UserRole;
  visibleGroups: number;
  visibleItems: number;
  groupDetails: Array<{
    groupName: string;
    visible: boolean;
    itemCount: number;
    items: Array<{
      name: string;
      visible: boolean;
      hasChildren: boolean;
      childCount: number;
    }>;
  }>;
}

/**
 * Test navigation filtering for a specific user role
 */
export const testNavigationForRole = (userRole: UserRole): NavigationTestResult => {
  let totalVisibleGroups = 0;
  let totalVisibleItems = 0;

  const canAccessNavItem = (navItem: NavItem): boolean => {
    if (navItem.roles && navItem.roles.length > 0) {
      return hasAnyRole(userRole, navItem.roles as UserRole[]);
    }
    return true; // If no roles specified, accessible to all
  };

  const countVisibleItems = (navItems: NavItem[]): number => {
    let count = 0;
    for (const item of navItems) {
      if (canAccessNavItem(item)) {
        count++;
        if (item.children) {
          count += countVisibleItems(item.children);
        }
      }
    }
    return count;
  };

  const groupDetails = routeGroups.map(group => {
    const canAccessGroup = !group.roles || group.roles.length === 0 ||
                          hasAnyRole(userRole, group.roles as UserRole[]);

    if (canAccessGroup) {
      totalVisibleGroups++;
    }

    const visibleItemCount = canAccessGroup ? countVisibleItems(group.children) : 0;
    totalVisibleItems += visibleItemCount;

    const items = group.children.map(item => ({
      name: item.name,
      visible: canAccessGroup && canAccessNavItem(item),
      hasChildren: !!item.children && item.children.length > 0,
      childCount: item.children ? countVisibleItems(item.children) : 0
    }));

    return {
      groupName: group.label,
      visible: canAccessGroup,
      itemCount: visibleItemCount,
      items
    };
  });

  return {
    role: userRole,
    visibleGroups: totalVisibleGroups,
    visibleItems: totalVisibleItems,
    groupDetails
  };
};

/**
 * Test navigation filtering for all roles
 */
export const testNavigationForAllRoles = (): NavigationTestResult[] => {
  const roles: UserRole[] = ['guest', 'operator', 'manager', 'administrator', 'ceo', 'developer'];
  return roles.map(role => testNavigationForRole(role));
};

/**
 * Print navigation test results to console (for debugging)
 */
export const logNavigationTests = () => {
  const results = testNavigationForAllRoles();

  console.group('🔐 Role-Based Navigation Test Results');

  results.forEach(result => {
    console.group(`👤 Role: ${result.role.toUpperCase()}`);
    console.log(`📊 Visible Groups: ${result.visibleGroups}`);
    console.log(`📋 Visible Items: ${result.visibleItems}`);

    result.groupDetails.forEach(group => {
      const icon = group.visible ? '✅' : '❌';
      console.log(`${icon} Group: ${group.groupName} (${group.itemCount} items)`);

      if (group.visible && group.items.length > 0) {
        group.items.forEach(item => {
          const itemIcon = item.visible ? '  ✓' : '  ✗';
          const childInfo = item.hasChildren ? ` (+${item.childCount} children)` : '';
          console.log(`${itemIcon} ${item.name}${childInfo}`);
        });
      }
    });

    console.groupEnd();
  });

  console.groupEnd();
};

/**
 * Verify role hierarchy is working correctly
 */
export const testRoleHierarchy = () => {
  console.group('🔐 Role Hierarchy Test');

  const testCases = [
    { user: 'operator', required: 'operator', expected: true },
    { user: 'operator', required: 'manager', expected: false },
    { user: 'manager', required: 'operator', expected: true },
    { user: 'manager', required: 'manager', expected: true },
    { user: 'manager', required: 'administrator', expected: false },
    { user: 'administrator', required: 'operator', expected: true },
    { user: 'administrator', required: 'manager', expected: true },
    { user: 'administrator', required: 'administrator', expected: true },
    { user: 'administrator', required: 'ceo', expected: false },
    { user: 'ceo', required: 'operator', expected: true },
    { user: 'ceo', required: 'administrator', expected: true },
    { user: 'ceo', required: 'ceo', expected: true },
    { user: 'ceo', required: 'developer', expected: false },
    { user: 'developer', required: 'operator', expected: true },
    { user: 'developer', required: 'manager', expected: true },
    { user: 'developer', required: 'administrator', expected: true },
    { user: 'developer', required: 'ceo', expected: true },
    { user: 'developer', required: 'developer', expected: true },
  ];

  let passedTests = 0;

  testCases.forEach(test => {
    const result = hasRoleAccess(test.user as UserRole, test.required as UserRole);
    const icon = result === test.expected ? '✅' : '❌';
    const status = result === test.expected ? 'PASS' : 'FAIL';

    console.log(`${icon} ${status}: ${test.user} -> ${test.required} = ${result}`);

    if (result === test.expected) {
      passedTests++;
    }
  });

  console.log(`\n📊 Test Results: ${passedTests}/${testCases.length} tests passed`);
  console.groupEnd();
};

/**
 * Development utility to run all tests
 */
export const runAllNavigationTests = () => {
  if (process.env.NODE_ENV === 'development') {
    console.clear();
    console.log('🚀 Running Role-Based Navigation Tests...\n');

    testRoleHierarchy();
    console.log(''); // Add spacing
    logNavigationTests();

    console.log('\n✨ Test complete! Check the results above.');
  }
};