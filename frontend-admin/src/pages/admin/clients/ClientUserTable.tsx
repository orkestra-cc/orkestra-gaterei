import { useMemo, useState } from 'react';
import { Card, Form, InputGroup, Spinner, Table } from 'react-bootstrap';
import { Link } from 'react-router';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import SubtleBadge from 'components/common/SubtleBadge';
import type { BadgeColor } from 'components/common/SubtleBadge';
import type { AdminClientUserItem } from 'store/api/userApi';

const roleColors: Record<string, BadgeColor> = {
  super_admin: 'danger',
  administrator: 'primary',
  developer: 'info',
  manager: 'warning',
  operator: 'success',
  guest: 'secondary',
};

interface Props {
  users: AdminClientUserItem[];
  isLoading: boolean;
  error: boolean;
}

const formatDate = (dateStr?: string | null) => {
  if (!dateStr) return '—';
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
};

const ClientUserTable: React.FC<Props> = ({ users, isLoading, error }) => {
  const [search, setSearch] = useState('');
  const [attachedFilter, setAttachedFilter] = useState<'all' | 'attached' | 'unattached'>('all');

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return users.filter((u) => {
      if (
        term &&
        !u.email.toLowerCase().includes(term) &&
        !(u.fullName ?? '').toLowerCase().includes(term) &&
        !(u.username ?? '').toLowerCase().includes(term)
      ) {
        return false;
      }
      if (attachedFilter === 'attached' && u.memberships.length === 0) return false;
      if (attachedFilter === 'unattached' && u.memberships.length > 0) return false;
      return true;
    });
  }, [users, search, attachedFilter]);

  if (error) {
    return (
      <Card>
        <Card.Body className="text-center text-danger py-5">
          Failed to load client users. You need the <code>system.users.admin</code>{' '}
          permission to view this page.
        </Card.Body>
      </Card>
    );
  }

  return (
    <Card>
      <Card.Header className="border-bottom border-200 px-4 py-3">
        <div className="d-flex flex-wrap gap-3 align-items-center justify-content-between">
          <h5 className="mb-0">Client users</h5>
          <div className="d-flex flex-wrap gap-2 align-items-center">
            <Form.Select
              size="sm"
              style={{ width: 'auto' }}
              value={attachedFilter}
              onChange={(e) =>
                setAttachedFilter(e.target.value as 'all' | 'attached' | 'unattached')
              }
            >
              <option value="all">All users</option>
              <option value="attached">Attached to a tenant</option>
              <option value="unattached">Unattached</option>
            </Form.Select>
            <InputGroup size="sm" style={{ width: 240 }}>
              <InputGroup.Text>
                <FontAwesomeIcon icon="search" />
              </InputGroup.Text>
              <Form.Control
                placeholder="Search by name, email, username"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </InputGroup>
          </div>
        </div>
      </Card.Header>
      <Card.Body className="p-0">
        {isLoading ? (
          <div className="text-center py-5">
            <Spinner animation="border" size="sm" />
          </div>
        ) : (
          <Table responsive size="sm" className="fs-10 mb-0 overflow-hidden">
            <thead className="bg-body-tertiary">
              <tr>
                <th className="pe-4 ps-3">User</th>
                <th>Role</th>
                <th>Tenants</th>
                <th>Status</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={5} className="text-center text-body-tertiary py-4">
                    No client users match the current filters.
                  </td>
                </tr>
              ) : (
                filtered.map((u) => (
                  <tr key={u.id}>
                    <td className="ps-3">
                      <Link
                        to={`/admin/clients/${u.id}`}
                        className="text-decoration-none text-body"
                      >
                        <div className="fw-semibold">{u.fullName || u.username || '—'}</div>
                        <div className="text-body-tertiary fs-11">{u.email}</div>
                      </Link>
                    </td>
                    <td>
                      <SubtleBadge bg={roleColors[u.role] ?? 'secondary'} pill>
                        {u.role}
                      </SubtleBadge>
                    </td>
                    <td>
                      {u.memberships.length === 0 ? (
                        <SubtleBadge bg="secondary" pill className="fs-11">
                          unattached
                        </SubtleBadge>
                      ) : (
                        <div className="d-flex flex-wrap gap-1">
                          {u.memberships.map((m) => (
                            <Link
                              key={m.tenantUUID}
                              to={`/admin/external-tenants/${m.tenantUUID}`}
                              className="text-decoration-none"
                              title={
                                m.isOwner
                                  ? `Owner of ${m.tenantName}`
                                  : `Member of ${m.tenantName}`
                              }
                            >
                              <SubtleBadge
                                bg={m.isOwner ? 'primary' : 'info'}
                                pill
                                className="fs-11"
                              >
                                {m.tenantName}
                                {m.isOwner && (
                                  <FontAwesomeIcon
                                    icon="crown"
                                    className="ms-1"
                                    style={{ fontSize: '0.7em' }}
                                  />
                                )}
                              </SubtleBadge>
                            </Link>
                          ))}
                        </div>
                      )}
                    </td>
                    <td>
                      {u.isActive ? (
                        <SubtleBadge bg="success" pill className="fs-11">
                          active
                        </SubtleBadge>
                      ) : (
                        <SubtleBadge bg="warning" pill className="fs-11">
                          disabled
                        </SubtleBadge>
                      )}
                      {!u.emailVerified && (
                        <SubtleBadge bg="secondary" pill className="fs-11 ms-1">
                          unverified
                        </SubtleBadge>
                      )}
                    </td>
                    <td>{formatDate(u.createdAt)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </Table>
        )}
      </Card.Body>
    </Card>
  );
};

export default ClientUserTable;
