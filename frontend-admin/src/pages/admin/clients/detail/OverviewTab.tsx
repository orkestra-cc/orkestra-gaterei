import { useState } from 'react';
import { Alert, Button, Card, Col, Form, Row, Spinner } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import SubtleBadge from 'components/common/SubtleBadge';
import {
  useUpdateClientUserAdminMutation,
  type AdminClientUserItem,
  type UpdateClientUserAdminInput,
} from 'store/api/userApi';

interface Props {
  user: AdminClientUserItem;
}

const ROLES = [
  'super_admin',
  'administrator',
  'developer',
  'manager',
  'operator',
  'guest',
] as const;

const formatDate = (iso?: string | null) => {
  if (!iso) return '—';
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

const OverviewTab: React.FC<Props> = ({ user }) => {
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<UpdateClientUserAdminInput>({
    fullName: user.fullName ?? '',
    username: user.username ?? '',
    role: user.role,
    isActive: user.isActive,
  });
  const [updateUser, { isLoading, error }] = useUpdateClientUserAdminMutation();
  const [success, setSuccess] = useState<string | null>(null);

  const onCancel = () => {
    setEditing(false);
    setSuccess(null);
    setForm({
      fullName: user.fullName ?? '',
      username: user.username ?? '',
      role: user.role,
      isActive: user.isActive,
    });
  };

  const onSave = async () => {
    setSuccess(null);
    try {
      await updateUser({ id: user.id, data: form }).unwrap();
      setSuccess('Profile updated.');
      setEditing(false);
    } catch {
      // RTK Query keeps the error in the hook state
    }
  };

  return (
    <Row className="g-3">
      <Col lg={7}>
        <Card className="shadow-none border h-100">
          <Card.Header className="border-bottom border-200 d-flex justify-content-between align-items-center">
            <h6 className="mb-0">Profile</h6>
            {!editing ? (
              <Button size="sm" variant="outline-primary" onClick={() => setEditing(true)}>
                <FontAwesomeIcon icon="edit" className="me-1" />
                Edit
              </Button>
            ) : (
              <div className="d-flex gap-2">
                <Button size="sm" variant="outline-secondary" onClick={onCancel}>
                  Cancel
                </Button>
                <Button size="sm" variant="primary" onClick={onSave} disabled={isLoading}>
                  {isLoading ? <Spinner size="sm" animation="border" /> : 'Save'}
                </Button>
              </div>
            )}
          </Card.Header>
          <Card.Body>
            {success && (
              <Alert variant="success" onClose={() => setSuccess(null)} dismissible>
                {success}
              </Alert>
            )}
            {error && (
              <Alert variant="danger">
                Failed to update profile. Check the form and try again.
              </Alert>
            )}
            <Row className="g-3">
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Full name</Form.Label>
                {editing ? (
                  <Form.Control
                    size="sm"
                    value={form.fullName ?? ''}
                    onChange={(e) => setForm((f) => ({ ...f, fullName: e.target.value }))}
                  />
                ) : (
                  <div>{user.fullName || '—'}</div>
                )}
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Username</Form.Label>
                {editing ? (
                  <Form.Control
                    size="sm"
                    value={form.username ?? ''}
                    onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
                  />
                ) : (
                  <div>{user.username || '—'}</div>
                )}
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Email</Form.Label>
                <div>
                  {user.email}{' '}
                  {!user.emailVerified && (
                    <SubtleBadge bg="secondary" pill className="fs-11 ms-1">
                      unverified
                    </SubtleBadge>
                  )}
                </div>
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Role</Form.Label>
                {editing ? (
                  <Form.Select
                    size="sm"
                    value={form.role ?? user.role}
                    onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))}
                  >
                    {ROLES.map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </Form.Select>
                ) : (
                  <SubtleBadge bg="info" pill>
                    {user.role}
                  </SubtleBadge>
                )}
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Status</Form.Label>
                {editing ? (
                  <Form.Check
                    type="switch"
                    id="active-switch"
                    label={form.isActive ? 'Active' : 'Disabled'}
                    checked={!!form.isActive}
                    onChange={(e) => setForm((f) => ({ ...f, isActive: e.target.checked }))}
                  />
                ) : user.isActive ? (
                  <SubtleBadge bg="success" pill>
                    active
                  </SubtleBadge>
                ) : (
                  <SubtleBadge bg="warning" pill>
                    disabled
                  </SubtleBadge>
                )}
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Last login</Form.Label>
                <div>{formatDate(user.lastLogin)}</div>
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">Created</Form.Label>
                <div>{formatDate(user.createdAt)}</div>
              </Col>
              <Col md={6}>
                <Form.Label className="text-body-tertiary fs-10 mb-1">User ID</Form.Label>
                <code className="fs-11">{user.id}</code>
              </Col>
            </Row>
          </Card.Body>
        </Card>
      </Col>

      <Col lg={5}>
        <Card className="shadow-none border h-100">
          <Card.Header className="border-bottom border-200">
            <h6 className="mb-0">OAuth providers</h6>
          </Card.Header>
          <Card.Body>
            {!user.providers || user.providers.length === 0 ? (
              <div className="text-body-tertiary fs-10">
                No OAuth provider linked. The user signs in with email + password.
              </div>
            ) : (
              <div className="d-flex flex-column gap-2">
                {user.providers.map((p) => (
                  <div
                    key={`${p.provider}-${p.email}`}
                    className="d-flex align-items-center gap-2 fs-10"
                  >
                    <SubtleBadge bg="primary" pill>
                      {p.provider}
                    </SubtleBadge>
                    <span className="text-body-tertiary">{p.email}</span>
                  </div>
                ))}
              </div>
            )}
          </Card.Body>
        </Card>
      </Col>
    </Row>
  );
};

export default OverviewTab;
