import { useState } from 'react';
import { Alert, Button, Card, Spinner, Table } from 'react-bootstrap';
import { Link } from 'react-router';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import SubtleBadge from 'components/common/SubtleBadge';
import { useRemoveOrgMemberAdminMutation } from 'store/api/tenantApi';
import type { AdminClientUserItem } from 'store/api/userApi';
import AttachToTenantModal from './AttachToTenantModal';

interface Props {
  user: AdminClientUserItem;
}

const MembershipsTab: React.FC<Props> = ({ user }) => {
  const [showAttach, setShowAttach] = useState(false);
  const [removeMember, { isLoading: removing }] = useRemoveOrgMemberAdminMutation();
  const [pendingRemove, setPendingRemove] = useState<string | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);

  const onDetach = async (tenantId: string) => {
    setRemoveError(null);
    setPendingRemove(tenantId);
    try {
      await removeMember({ tenantId, userUUID: user.id }).unwrap();
    } catch (err) {
      // Surface the server message so the admin knows why detach was refused
      // (e.g. last owner of the tenant).
      const msg =
        (err as { data?: { detail?: string; message?: string } }).data?.detail ??
        (err as { data?: { detail?: string; message?: string } }).data?.message ??
        'Failed to detach. The user may be the last owner of that tenant.';
      setRemoveError(msg);
    } finally {
      setPendingRemove(null);
    }
  };

  return (
    <>
      <Card className="shadow-none border">
        <Card.Header className="border-bottom border-200 d-flex justify-content-between align-items-center">
          <h6 className="mb-0">Tenant memberships</h6>
          <Button size="sm" variant="primary" onClick={() => setShowAttach(true)}>
            <FontAwesomeIcon icon="plus" className="me-1" />
            Attach to tenant
          </Button>
        </Card.Header>
        <Card.Body className="p-0">
          {removeError && (
            <Alert variant="danger" className="m-3 mb-0">
              {removeError}
            </Alert>
          )}
          {user.memberships.length === 0 ? (
            <div className="text-body-tertiary fs-10 py-5 text-center">
              This user is not attached to any tenant yet. Use “Attach to tenant” to link them
              to one of the existing external tenants.
            </div>
          ) : (
            <Table responsive size="sm" className="fs-10 mb-0">
              <thead className="bg-body-tertiary">
                <tr>
                  <th className="ps-3">Tenant</th>
                  <th>Roles</th>
                  <th>Owner</th>
                  <th className="pe-3 text-end">Actions</th>
                </tr>
              </thead>
              <tbody>
                {user.memberships.map((m) => (
                  <tr key={m.tenantUUID}>
                    <td className="ps-3 align-middle">
                      <Link
                        to={`/admin/external-tenants/${m.tenantUUID}`}
                        className="text-decoration-none fw-semibold"
                      >
                        {m.tenantName}
                      </Link>
                      {m.tenantSlug && (
                        <span className="text-body-tertiary ms-2">
                          <code className="fs-11">{m.tenantSlug}</code>
                        </span>
                      )}
                    </td>
                    <td className="align-middle">
                      <div className="d-flex flex-wrap gap-1">
                        {(m.roles ?? []).map((r) => (
                          <SubtleBadge key={r} bg="info" pill className="fs-11">
                            {r}
                          </SubtleBadge>
                        ))}
                      </div>
                    </td>
                    <td className="align-middle">
                      {m.isOwner ? (
                        <SubtleBadge bg="primary" pill>
                          <FontAwesomeIcon icon="crown" className="me-1" /> owner
                        </SubtleBadge>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="pe-3 text-end align-middle">
                      <Button
                        size="sm"
                        variant="outline-danger"
                        onClick={() => onDetach(m.tenantUUID)}
                        disabled={removing && pendingRemove === m.tenantUUID}
                      >
                        {removing && pendingRemove === m.tenantUUID ? (
                          <Spinner size="sm" animation="border" />
                        ) : (
                          'Detach'
                        )}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </Card.Body>
      </Card>

      {showAttach && (
        <AttachToTenantModal
          show={showAttach}
          onHide={() => setShowAttach(false)}
          userUUID={user.id}
          existingTenantIds={user.memberships.map((m) => m.tenantUUID)}
        />
      )}
    </>
  );
};

export default MembershipsTab;
