import { useState } from 'react';
import { Alert, Spinner, Table } from 'react-bootstrap';
import { toast } from 'react-toastify';
import SubtleBadge from 'components/common/SubtleBadge';
import type { Org } from 'store/api/tenantApi';
import {
  useListOrgMembersAdminQuery,
  useRemoveOrgMemberAdminMutation,
} from 'store/api/tenantApi';
import { Button } from 'react-bootstrap';
import AttachMemberModal from './AttachMemberModal';

interface Props {
  org: Org;
}

/**
 * Members tab — mirrors the MembersTab from the legacy TenantDetailModal.
 * Role assignments live on the Role Management page; this tab only shows
 * current memberships and allows removing non-owner rows.
 */
const MembersTab: React.FC<Props> = ({ org }) => {
  const { data, isLoading, error } = useListOrgMembersAdminQuery(org.id);
  const [removeMember] = useRemoveOrgMemberAdminMutation();
  const [attachOpen, setAttachOpen] = useState(false);

  const onRemove = async (userUUID: string) => {
    try {
      await removeMember({ tenantId: org.id, userUUID }).unwrap();
      toast.success('Member removed');
    } catch (err: unknown) {
      toast.error('Remove failed: ' + extractError(err));
    }
  };

  if (isLoading) {
    return (
      <div className="text-center py-4">
        <Spinner animation="border" size="sm" />
      </div>
    );
  }
  if (error) {
    return (
      <Alert variant="danger" className="fs-10">
        Failed to load members.
      </Alert>
    );
  }

  const members = data?.members ?? [];

  return (
    <>
      <div className="d-flex justify-content-between align-items-start mb-2 gap-3">
        <Alert variant="info" className="fs-10 py-2 mb-0 flex-grow-1">
          Direct-attach a user as a member, or remove non-owner rows. Custom
          role assignments still live on the{' '}
          <a href="/admin/roles">Role Management page</a>.
        </Alert>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setAttachOpen(true)}
          className="flex-shrink-0"
        >
          Attach Member
        </Button>
      </div>
      <Table size="sm" className="fs-10 mb-0">
        <thead className="bg-body-tertiary">
          <tr>
            <th>Email</th>
            <th>User UUID</th>
            <th>Roles</th>
            <th>Joined</th>
            <th>Owner</th>
            <th className="text-end">Actions</th>
          </tr>
        </thead>
        <tbody>
          {members.map((m) => (
            <tr key={m.id} className="align-middle">
              <td>{m.email || <span className="text-muted">—</span>}</td>
              <td className="font-monospace fs-11">{m.userUUID}</td>
              <td>{m.roles.join(', ') || '—'}</td>
              <td className="text-muted">
                {m.joinedAt ? new Date(m.joinedAt).toLocaleDateString() : '—'}
              </td>
              <td>
                {m.isOwner && (
                  <SubtleBadge bg="primary" pill>
                    owner
                  </SubtleBadge>
                )}
              </td>
              <td className="text-end">
                {!m.isOwner && (
                  <Button
                    variant="link"
                    size="sm"
                    className="p-0 text-danger text-decoration-none"
                    onClick={() => onRemove(m.userUUID)}
                  >
                    Remove
                  </Button>
                )}
              </td>
            </tr>
          ))}
          {members.length === 0 && (
            <tr>
              <td colSpan={6} className="text-center text-muted py-3">
                No members yet.
              </td>
            </tr>
          )}
        </tbody>
      </Table>
      <AttachMemberModal
        org={org}
        show={attachOpen}
        onHide={() => setAttachOpen(false)}
      />
    </>
  );
};

function extractError(err: unknown): string {
  if (err && typeof err === 'object' && 'data' in err) {
    const data = (err as { data?: { detail?: string; title?: string } }).data;
    return data?.detail || data?.title || 'unknown error';
  }
  return String(err);
}

export default MembersTab;
