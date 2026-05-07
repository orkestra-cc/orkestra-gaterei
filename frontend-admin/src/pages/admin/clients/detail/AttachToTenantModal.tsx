import { useMemo, useState } from 'react';
import { Alert, Button, Form, Modal, Spinner } from 'react-bootstrap';
import {
  useAttachOrgMemberAdminMutation,
  useListAllOrgsAdminQuery,
} from 'store/api/tenantApi';

interface Props {
  show: boolean;
  onHide: () => void;
  userUUID: string;
  existingTenantIds: string[];
}

const ROLE_OPTIONS = [
  { value: 'org_owner', label: 'org_owner — full control of the tenant' },
  { value: 'org_admin', label: 'org_admin — manage members + settings' },
  { value: 'org_member', label: 'org_member — standard member' },
] as const;

const AttachToTenantModal: React.FC<Props> = ({
  show,
  onHide,
  userUUID,
  existingTenantIds,
}) => {
  const [search, setSearch] = useState('');
  const [tenantId, setTenantId] = useState('');
  const [role, setRole] = useState<string>('org_member');
  const [isOwner, setIsOwner] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const { data, isLoading } = useListAllOrgsAdminQuery({ kind: 'external' });
  const [attach, { isLoading: attaching }] = useAttachOrgMemberAdminMutation();

  const candidates = useMemo(() => {
    const orgs = data?.tenants ?? [];
    const term = search.trim().toLowerCase();
    return orgs
      .filter((o) => !o.deletedAt)
      .filter((o) => !existingTenantIds.includes(o.id))
      .filter((o) =>
        term
          ? o.name.toLowerCase().includes(term) || o.slug.toLowerCase().includes(term)
          : true,
      )
      .slice(0, 25);
  }, [data, search, existingTenantIds]);

  const onSubmit = async () => {
    setSubmitError(null);
    if (!tenantId) {
      setSubmitError('Pick a tenant first.');
      return;
    }
    try {
      await attach({
        tenantId,
        body: { userUuid: userUUID, role, isOwner: isOwner || undefined },
      }).unwrap();
      onHide();
    } catch (err) {
      const msg =
        (err as { data?: { detail?: string; message?: string } }).data?.detail ??
        (err as { data?: { detail?: string; message?: string } }).data?.message ??
        'Failed to attach. The user may already be a member or you may lack permission.';
      setSubmitError(msg);
    }
  };

  return (
    <Modal show={show} onHide={onHide} centered>
      <Modal.Header closeButton>
        <Modal.Title>Attach to tenant</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {submitError && <Alert variant="danger">{submitError}</Alert>}

        <Form.Group className="mb-3">
          <Form.Label>Search external tenants</Form.Label>
          <Form.Control
            size="sm"
            placeholder="Type to filter by name or slug"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </Form.Group>

        <Form.Group className="mb-3">
          <Form.Label>Pick a tenant</Form.Label>
          {isLoading ? (
            <div className="text-center py-3">
              <Spinner size="sm" animation="border" />
            </div>
          ) : (
            <Form.Select
              size="sm"
              value={tenantId}
              onChange={(e) => setTenantId(e.target.value)}
            >
              <option value="">— select —</option>
              {candidates.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.name} ({o.slug})
                </option>
              ))}
            </Form.Select>
          )}
          {!isLoading && candidates.length === 0 && (
            <div className="text-body-tertiary fs-10 mt-1">
              No matching external tenants. The user may already be in every tenant
              you have, or no external tenants exist yet.
            </div>
          )}
        </Form.Group>

        <Form.Group className="mb-3">
          <Form.Label>Role</Form.Label>
          <Form.Select size="sm" value={role} onChange={(e) => setRole(e.target.value)}>
            {ROLE_OPTIONS.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </Form.Select>
        </Form.Group>

        <Form.Check
          type="switch"
          id="attach-owner-switch"
          label="Mark as tenant owner"
          checked={isOwner}
          onChange={(e) => setIsOwner(e.target.checked)}
        />
      </Modal.Body>
      <Modal.Footer>
        <Button variant="outline-secondary" size="sm" onClick={onHide}>
          Cancel
        </Button>
        <Button
          variant="primary"
          size="sm"
          onClick={onSubmit}
          disabled={attaching || !tenantId}
        >
          {attaching ? <Spinner size="sm" animation="border" /> : 'Attach'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default AttachToTenantModal;
