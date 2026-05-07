import { useMemo, useState } from 'react';
import { Button, Card, Col, Row } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import type { IconProp } from '@fortawesome/fontawesome-svg-core';
import {
  faUsers,
  faCircleCheck,
  faLink,
  faUserSlash,
} from '@fortawesome/free-solid-svg-icons';
import CountUp from 'react-countup';
import { useListClientUsersAdminQuery } from 'store/api/userApi';
import ClientUserTable from './ClientUserTable';
import CreateClientUserModal from './CreateClientUserModal';

interface StatCardProps {
  title: string;
  value: number;
  icon: IconProp;
  accent: 'primary' | 'success' | 'info' | 'warning' | 'secondary';
  footnote?: React.ReactNode;
}

const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  icon,
  accent,
  footnote,
}) => (
  <Card className="h-100 shadow-none border">
    <Card.Body>
      <div className="d-flex justify-content-between align-items-start">
        <div>
          <h6 className="text-body-tertiary fs-10 text-uppercase mb-2">{title}</h6>
          <h3 className="fw-normal text-body mb-0">
            <CountUp start={0} end={value} duration={1.5} separator="," />
          </h3>
        </div>
        <div
          className={`d-flex align-items-center justify-content-center rounded-circle bg-${accent}-subtle`}
          style={{ width: 48, height: 48 }}
        >
          <FontAwesomeIcon icon={icon} className={`fs-5 text-${accent}`} />
        </div>
      </div>
      {footnote && (
        <div className="fs-10 text-body-tertiary mt-3">{footnote}</div>
      )}
    </Card.Body>
  </Card>
);

/**
 * Client Management page (Tier-2 external users).
 *
 * Lists rows from the `client_users` collection joined with each user's
 * tenant memberships. Row click navigates to /admin/clients/:userId
 * (user detail); tenant badges link to /admin/external-tenants/:tenantId
 * (org detail).
 */
const ClientManagementPage: React.FC = () => {
  const [showCreate, setShowCreate] = useState(false);
  // Initial list pulls a generous page size so the dashboard stats are
  // accurate without a separate count call. If the population grows past
  // a few hundred we'll add real pagination on the table.
  const { data, isLoading, error } = useListClientUsersAdminQuery({ pageSize: 200 });

  const users = useMemo(() => data?.users ?? [], [data]);

  const stats = useMemo(() => {
    const total = users.length;
    const active = users.filter((u) => u.isActive).length;
    const attached = users.filter((u) => u.memberships.length > 0).length;
    const unattached = total - attached;
    return { total, active, attached, unattached };
  }, [users]);

  return (
    <>
      <div className="d-flex justify-content-end mb-3">
        <Button variant="primary" size="sm" onClick={() => setShowCreate(true)}>
          <FontAwesomeIcon icon="plus" className="me-1" />
          Invite or create user
        </Button>
      </div>
      <Row className="g-3 mb-4">
        <Col md={6} xl={3}>
          <StatCard
            title="Total client users"
            value={stats.total}
            icon={faUsers}
            accent="primary"
            footnote={
              data && data.total > stats.total
                ? `${stats.total} of ${data.total} loaded`
                : 'All users loaded'
            }
          />
        </Col>
        <Col md={6} xl={3}>
          <StatCard
            title="Active"
            value={stats.active}
            icon={faCircleCheck}
            accent="success"
            footnote={
              stats.total > 0
                ? `${Math.round((stats.active / stats.total) * 100)}% of total`
                : '—'
            }
          />
        </Col>
        <Col md={6} xl={3}>
          <StatCard
            title="Attached"
            value={stats.attached}
            icon={faLink}
            accent="info"
            footnote="Users with at least one tenant membership"
          />
        </Col>
        <Col md={6} xl={3}>
          <StatCard
            title="Unattached"
            value={stats.unattached}
            icon={faUserSlash}
            accent="warning"
            footnote="Self-registered, not yet linked to a tenant"
          />
        </Col>
      </Row>

      <ClientUserTable users={users} isLoading={isLoading} error={!!error} />

      {showCreate && (
        <CreateClientUserModal show={showCreate} onHide={() => setShowCreate(false)} />
      )}
    </>
  );
};

export default ClientManagementPage;
