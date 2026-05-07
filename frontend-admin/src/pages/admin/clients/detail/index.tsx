import { Suspense, lazy, useState } from 'react';
import { Link, Navigate, useParams, useSearchParams } from 'react-router';
import { Alert, Breadcrumb, Button, Card, Nav, Spinner, Tab } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import SubtleBadge from 'components/common/SubtleBadge';
import { useGetClientUserAdminQuery } from 'store/api/userApi';
import DeleteClientUserModal from './DeleteClientUserModal';

const OverviewTab = lazy(() => import('./OverviewTab'));
const MembershipsTab = lazy(() => import('./MembershipsTab'));

const TAB_KEYS = ['overview', 'memberships'] as const;
type TabKey = (typeof TAB_KEYS)[number];
const DEFAULT_TAB: TabKey = 'overview';

// URL-tabs convention: persist the active tab to ?tab= so deep links and
// reloads land on the same view. Unknown values fall back to overview.
function readTab(param: string | null): TabKey {
  const candidate = (param ?? DEFAULT_TAB) as TabKey;
  return TAB_KEYS.includes(candidate) ? candidate : DEFAULT_TAB;
}

const ClientUserDetailPage: React.FC = () => {
  const { userId } = useParams<{ userId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = readTab(searchParams.get('tab'));
  const [showDelete, setShowDelete] = useState(false);

  const { data: user, isLoading, error } = useGetClientUserAdminQuery(userId ?? '', {
    skip: !userId,
  });

  if (!userId) {
    return <Navigate to="/admin/clients" replace />;
  }

  if (isLoading) {
    return (
      <div className="text-center py-5">
        <Spinner animation="border" size="sm" />
      </div>
    );
  }

  if (error || !user) {
    return (
      <Alert variant="danger">
        Client user not found or you lack permission to view it.{' '}
        <Link to="/admin/clients">Back to clients</Link>
      </Alert>
    );
  }

  const onTabChange = (key: string | null) => {
    const next = readTab(key);
    const sp = new URLSearchParams(searchParams);
    if (next === DEFAULT_TAB) sp.delete('tab');
    else sp.set('tab', next);
    setSearchParams(sp, { replace: true });
  };

  return (
    <>
      <Breadcrumb className="mb-3 fs-10">
        <Breadcrumb.Item linkAs={Link} linkProps={{ to: '/admin/clients' }}>
          Clients
        </Breadcrumb.Item>
        <Breadcrumb.Item active>{user.fullName || user.email}</Breadcrumb.Item>
      </Breadcrumb>

      <Card className="mb-3 shadow-none border">
        <Card.Body className="d-flex justify-content-between align-items-start flex-wrap gap-3">
          <div>
            <h3 className="fw-normal mb-1">
              <FontAwesomeIcon icon="user" className="text-primary me-2" />
              {user.fullName || user.username || user.email}
            </h3>
            <div className="d-flex align-items-center gap-2 flex-wrap fs-10 text-muted">
              <span>{user.email}</span>
              <SubtleBadge bg="info" pill>
                {user.role}
              </SubtleBadge>
              {user.isActive ? (
                <SubtleBadge bg="success" pill>
                  active
                </SubtleBadge>
              ) : (
                <SubtleBadge bg="warning" pill>
                  disabled
                </SubtleBadge>
              )}
              {!user.emailVerified && (
                <SubtleBadge bg="secondary" pill>
                  unverified
                </SubtleBadge>
              )}
              <SubtleBadge bg="primary" pill>
                Tier-2 client
              </SubtleBadge>
            </div>
          </div>
          <div className="d-flex flex-column align-items-end gap-2">
            <code className="fs-11 text-muted">{user.id}</code>
            <Button size="sm" variant="outline-danger" onClick={() => setShowDelete(true)}>
              <FontAwesomeIcon icon="trash" className="me-1" />
              Delete user
            </Button>
          </div>
        </Card.Body>
      </Card>

      <Card className="shadow-none border">
        <Tab.Container activeKey={tab} onSelect={onTabChange}>
          <Card.Header className="border-bottom border-200 p-0">
            <Nav variant="tabs" className="fs-10">
              <Nav.Item>
                <Nav.Link eventKey="overview">Overview</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="memberships">
                  Memberships
                  {user.memberships.length > 0 && (
                    <SubtleBadge bg="secondary" pill className="ms-2 fs-11">
                      {user.memberships.length}
                    </SubtleBadge>
                  )}
                </Nav.Link>
              </Nav.Item>
            </Nav>
          </Card.Header>
          <Card.Body>
            <Suspense
              fallback={
                <div className="text-center py-4">
                  <Spinner animation="border" size="sm" />
                </div>
              }
            >
              <Tab.Content>
                <Tab.Pane eventKey="overview">
                  <OverviewTab user={user} />
                </Tab.Pane>
                <Tab.Pane eventKey="memberships">
                  <MembershipsTab user={user} />
                </Tab.Pane>
              </Tab.Content>
            </Suspense>
          </Card.Body>
        </Tab.Container>
      </Card>

      {showDelete && (
        <DeleteClientUserModal
          show={showDelete}
          onHide={() => setShowDelete(false)}
          user={user}
        />
      )}
    </>
  );
};

export default ClientUserDetailPage;
