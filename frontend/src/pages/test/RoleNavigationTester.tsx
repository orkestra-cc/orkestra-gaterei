/**
 * Development component for testing role-based navigation
 * Only available in development environment
 */

import React, { useState } from 'react';
import { Card, Form, Row, Col, Badge, ListGroup, Alert } from 'react-bootstrap';
import { useRoleBasedNavigation } from 'hooks/useRoleBasedNavigation';
import { useCurrentUser } from 'hooks/auth/useAuthRTK';
import { UserRole, extractUserRole } from 'utils/roleUtils';
import routeGroups, { NavItem, RouteGroup } from 'routes/siteMaps';

// Mock user data for testing different roles
const mockUsers: Record<UserRole, any> = {
  guest: {
    user_id: '1',
    email: 'guest@example.com',
    full_name: 'Utente Ospite',
    role: 'guest',
    permissions: ['profile:read']
  },
  operator: {
    user_id: '2',
    email: 'operator@example.com',
    full_name: 'Giovanni Operatore',
    role: 'operator',
    permissions: ['task:execute', 'profile:read']
  },
  manager: {
    user_id: '3',
    email: 'manager@example.com',
    full_name: 'Maria Manager',
    role: 'manager',
    permissions: ['task:execute', 'task:assign', 'team:view', 'profile:read']
  },
  administrator: {
    user_id: '4',
    email: 'admin@example.com',
    full_name: 'Paolo Amministratore',
    role: 'administrator',
    permissions: ['task:execute', 'task:assign', 'team:view', 'fleet:manage', 'reports:view', 'profile:read', 'user:manage', 'system:settings']
  },
  ceo: {
    user_id: '5',
    email: 'ceo@example.com',
    full_name: 'Andrea CEO',
    role: 'ceo',
    permissions: ['*'] // All permissions
  },
  developer: {
    user_id: '6',
    email: 'developer@example.com',
    full_name: 'Marco Sviluppatore',
    role: 'developer',
    permissions: ['*'] // All permissions
  }
};

interface RoleNavigationTesterProps {
  // This component doesn't require any props
}

const RoleNavigationTester: React.FC<RoleNavigationTesterProps> = () => {
  const [selectedRole, setSelectedRole] = useState<UserRole>('operator');
  const { user: realUser, isAuthenticated } = useCurrentUser();
  const realUserRole = extractUserRole(realUser);

  // Get real filtered navigation using the actual hook
  const { filteredNavigation: realFilteredNavigation } = useRoleBasedNavigation(routeGroups);

  // Mock the filtered navigation for the selected role for testing
  const mockFilteredNavigation = React.useMemo(() => {

    // Simple mock implementation of role-based filtering
    const canAccessNavItem = (item: NavItem): boolean => {
      if (!item.roles || item.roles.length === 0) return true;
      return item.roles.includes(selectedRole);
    };

    const canAccessGroup = (group: RouteGroup): boolean => {
      if (!group.roles || group.roles.length === 0) return true;
      return group.roles.includes(selectedRole);
    };

    const filterNavItems = (items: NavItem[]): NavItem[] => {
      return items
        .filter(canAccessNavItem)
        .map(item => ({
          ...item,
          children: item.children ? filterNavItems(item.children) : undefined
        }))
        .filter(item => !item.children || item.children.length > 0);
    };

    return routeGroups
      .filter(canAccessGroup)
      .map(group => ({
        ...group,
        children: filterNavItems(group.children)
      }))
      .filter(group => group.children.length > 0);
  }, [selectedRole]);

  const renderNavItem = (item: NavItem, depth: number = 0) => {
    const indentStyle = { paddingLeft: `${depth * 20 + 10}px` };

    return (
      <div key={item.name}>
        <ListGroup.Item style={indentStyle} className="d-flex justify-content-between align-items-center">
          <div>
            {item.icon && <i className={`fas fa-${item.icon} me-2`}></i>}
            {item.name}
            {item.to && <small className="text-muted ms-2">({item.to})</small>}
          </div>
          <div>
            {item.roles && (
              <Badge bg="secondary" className="me-2">
                {item.roles.join(', ')}
              </Badge>
            )}
            {item.permissions && (
              <Badge bg="info">
                {item.permissions.join(', ')}
              </Badge>
            )}
          </div>
        </ListGroup.Item>
        {item.children && item.children.map(child => renderNavItem(child, depth + 1))}
      </div>
    );
  };

  // Don't render in production
  if (process.env.NODE_ENV !== 'development') {
    return null;
  }

  return (
    <div className="p-4">
      <h2>🔐 Role-Based Navigation Tester</h2>
      <p className="text-muted">
        This tool helps you test how navigation appears for different user roles.
        Only visible in development mode.
      </p>

      <Row className="mb-4">
        <Col>
          <Alert variant="info">
            <div className="d-flex justify-content-between align-items-center">
              <div>
                <strong>Current Real User:</strong> {' '}
                {isAuthenticated ? (
                  <>
                    <Badge bg="success">{realUserRole || 'Unknown Role'}</Badge>
                    <span className="ms-2 text-muted">({realUser?.fullName || realUser?.email || realUser?.id})</span>
                  </>
                ) : (
                  <Badge bg="danger">Not Authenticated</Badge>
                )}
              </div>
              <div>
                <small>Real navigation groups: {realFilteredNavigation.length}</small>
              </div>
            </div>
          </Alert>
        </Col>
      </Row>

      <Row>
        <Col md={4}>
          <Card>
            <Card.Header>
              <h5>Test Role Navigation</h5>
              <small className="text-muted">Simulate different user roles</small>
            </Card.Header>
            <Card.Body>
              <Form>
                <Form.Group>
                  <Form.Label>Current Role:</Form.Label>
                  <Form.Select
                    value={selectedRole}
                    onChange={(e) => setSelectedRole(e.target.value as UserRole)}
                  >
                    <option value="guest">Ospite</option>
                    <option value="operator">Operatore</option>
                    <option value="manager">Manager</option>
                    <option value="administrator">Amministratore</option>
                    <option value="ceo">CEO</option>
                    <option value="developer">Sviluppatore</option>
                  </Form.Select>
                </Form.Group>
              </Form>

              <hr />

              <div>
                <h6>Current User Details:</h6>
                <div>
                  <strong>Role:</strong> <Badge bg="primary">{selectedRole}</Badge>
                </div>
                <div>
                  <strong>Email:</strong> {mockUsers[selectedRole].email}
                </div>
                <div>
                  <strong>Permissions:</strong>
                  <div>
                    {mockUsers[selectedRole].permissions.map((perm: string) => (
                      <Badge key={perm} bg="secondary" className="me-1 mt-1">
                        {perm}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>
            </Card.Body>
          </Card>
        </Col>

        <Col md={8}>
          <Card>
            <Card.Header>
              <h5>Filtered Navigation for {selectedRole.toUpperCase()}</h5>
              <Badge bg="info">
                {mockFilteredNavigation.length} visible group(s)
              </Badge>
            </Card.Header>
            <Card.Body style={{ maxHeight: '600px', overflowY: 'auto' }}>
              {mockFilteredNavigation.length === 0 ? (
                <div className="text-center text-muted p-4">
                  <i className="fas fa-ban fa-3x mb-3"></i>
                  <p>No navigation items visible for this role.</p>
                </div>
              ) : (
                mockFilteredNavigation.map((group) => (
                  <div key={group.label} className="mb-4">
                    <div className="d-flex justify-content-between align-items-center mb-2">
                      <h6 className="text-primary mb-0">
                        📂 {group.label.toUpperCase()}
                      </h6>
                      <div>
                        {group.roles && (
                          <Badge bg="warning" text="dark">
                            Requires: {group.roles.join(', ')}
                          </Badge>
                        )}
                      </div>
                    </div>
                    <ListGroup variant="flush">
                      {group.children.map(item => renderNavItem(item))}
                    </ListGroup>
                  </div>
                ))
              )}
            </Card.Body>
          </Card>
        </Col>
      </Row>

      <Row className="mt-4">
        <Col>
          <Card>
            <Card.Header>
              <h6>Role Hierarchy Summary</h6>
            </Card.Header>
            <Card.Body>
              <p>
                <Badge bg="primary">developer</Badge> {'>'}
                <Badge bg="danger">ceo</Badge> {'>'}
                <Badge bg="warning" text="dark">administrator</Badge> {'>'}
                <Badge bg="info">manager</Badge> {'>'}
                <Badge bg="success">operator</Badge> {'>'}
                <Badge bg="secondary">guest</Badge>
              </p>
              <small className="text-muted">
                I ruoli superiori ereditano l'accesso dai ruoli inferiori. Ad esempio, un administrator
                può accedere a tutte le funzionalità di manager, operator e guest.
              </small>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default RoleNavigationTester;