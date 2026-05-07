import { useState } from 'react';
import { Alert, Button, Col, Form, Modal, Row, Spinner } from 'react-bootstrap';
import { useNavigate } from 'react-router';
import {
  useCreateClientUserAdminMutation,
  type CreateClientUserAdminInput,
} from 'store/api/userApi';

interface Props {
  show: boolean;
  onHide: () => void;
}

const ROLES = [
  'super_admin',
  'administrator',
  'developer',
  'manager',
  'operator',
  'guest',
] as const;

// Random 16-char temp password helper. Mixes case + digits + a couple of
// safe symbols so the live policy passes even with default complexity.
const generateTempPassword = () => {
  const chars =
    'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#$%';
  const out: string[] = [];
  const cryptoObj = window.crypto;
  for (let i = 0; i < 16; i += 1) {
    const buf = new Uint32Array(1);
    cryptoObj.getRandomValues(buf);
    out.push(chars[buf[0] % chars.length]);
  }
  return out.join('');
};

const CreateClientUserModal: React.FC<Props> = ({ show, onHide }) => {
  const navigate = useNavigate();
  const [form, setForm] = useState<CreateClientUserAdminInput>({
    email: '',
    fullName: '',
    username: '',
    phone: '',
    role: 'operator',
    password: generateTempPassword(),
  });
  const [error, setError] = useState<string | null>(null);
  const [createUser, { isLoading }] = useCreateClientUserAdminMutation();

  const onSubmit = async () => {
    setError(null);
    if (!form.email || !form.fullName || !form.password) {
      setError('Email, full name, and password are required.');
      return;
    }
    try {
      const created = await createUser(form).unwrap();
      onHide();
      navigate(`/admin/clients/${created.id}`);
    } catch (err) {
      const msg =
        (err as { data?: { detail?: string; message?: string } }).data?.detail ??
        (err as { data?: { detail?: string; message?: string } }).data?.message ??
        'Failed to create client user.';
      setError(msg);
    }
  };

  return (
    <Modal show={show} onHide={onHide} centered size="lg">
      <Modal.Header closeButton>
        <Modal.Title>New client user</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {error && <Alert variant="danger">{error}</Alert>}
        <p className="text-body-tertiary fs-10 mb-3">
          The new user is created in the <code>client_users</code> tier with email
          marked as verified. Share the temp password with them out-of-band; they
          can change it once logged in.
        </p>

        <Row className="g-3">
          <Col md={6}>
            <Form.Label className="fs-10">Email *</Form.Label>
            <Form.Control
              size="sm"
              type="email"
              value={form.email}
              onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
            />
          </Col>
          <Col md={6}>
            <Form.Label className="fs-10">Full name *</Form.Label>
            <Form.Control
              size="sm"
              value={form.fullName}
              onChange={(e) => setForm((f) => ({ ...f, fullName: e.target.value }))}
            />
          </Col>
          <Col md={6}>
            <Form.Label className="fs-10">Username</Form.Label>
            <Form.Control
              size="sm"
              value={form.username ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
            />
          </Col>
          <Col md={6}>
            <Form.Label className="fs-10">Phone (E.164)</Form.Label>
            <Form.Control
              size="sm"
              placeholder="+393331234567"
              value={form.phone ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))}
            />
          </Col>
          <Col md={6}>
            <Form.Label className="fs-10">System role</Form.Label>
            <Form.Select
              size="sm"
              value={form.role}
              onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))}
            >
              {ROLES.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </Form.Select>
            <Form.Text className="text-body-tertiary fs-11">
              Tier-2 client users typically default to <code>operator</code>. Tenant-
              scoped roles (org_owner / org_admin / org_member) are assigned via
              "Attach to tenant" once the user exists.
            </Form.Text>
          </Col>
          <Col md={6}>
            <Form.Label className="fs-10">Temporary password *</Form.Label>
            <div className="d-flex gap-2">
              <Form.Control
                size="sm"
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
              />
              <Button
                size="sm"
                variant="outline-secondary"
                type="button"
                onClick={() =>
                  setForm((f) => ({ ...f, password: generateTempPassword() }))
                }
              >
                Regenerate
              </Button>
            </div>
            <Form.Text className="text-body-tertiary fs-11">
              Validated against the live password policy.
            </Form.Text>
          </Col>
        </Row>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="outline-secondary" size="sm" onClick={onHide}>
          Cancel
        </Button>
        <Button variant="primary" size="sm" onClick={onSubmit} disabled={isLoading}>
          {isLoading ? <Spinner size="sm" animation="border" /> : 'Create user'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default CreateClientUserModal;
