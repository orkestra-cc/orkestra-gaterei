import { useState } from 'react';
import { Alert, Button, ButtonGroup, Col, Form, Modal, Row, Spinner } from 'react-bootstrap';
import { useNavigate } from 'react-router';
import {
  useCreateClientUserAdminMutation,
  useInviteClientUserAdminMutation,
  type CreateClientUserAdminInput,
  type InviteClientUserAdminInput,
} from 'store/api/userApi';

interface Props {
  show: boolean;
  onHide: () => void;
}

type Mode = 'invite' | 'create';

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
  const [mode, setMode] = useState<Mode>('invite');
  const [form, setForm] = useState<CreateClientUserAdminInput & { inviterName?: string }>({
    email: '',
    fullName: '',
    username: '',
    phone: '',
    role: 'operator',
    password: generateTempPassword(),
    inviterName: '',
  });
  const [error, setError] = useState<string | null>(null);
  const [createUser, { isLoading: creating }] = useCreateClientUserAdminMutation();
  const [inviteUser, { isLoading: inviting }] = useInviteClientUserAdminMutation();
  const isLoading = creating || inviting;

  const onSubmit = async () => {
    setError(null);
    if (!form.email || !form.fullName) {
      setError('Email and full name are required.');
      return;
    }
    if (mode === 'create' && !form.password) {
      setError('A temporary password is required when "Create with password" is selected.');
      return;
    }
    try {
      let createdId: string;
      if (mode === 'invite') {
        const payload: InviteClientUserAdminInput = {
          email: form.email,
          fullName: form.fullName,
          username: form.username,
          phone: form.phone,
          role: form.role,
          inviterName: form.inviterName,
        };
        const created = await inviteUser(payload).unwrap();
        createdId = created.id;
      } else {
        const created = await createUser({
          email: form.email,
          fullName: form.fullName,
          username: form.username,
          phone: form.phone,
          role: form.role,
          password: form.password,
        }).unwrap();
        createdId = created.id;
      }
      onHide();
      navigate(`/admin/clients/${createdId}`);
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

        <div className="mb-3">
          <ButtonGroup size="sm" className="w-100">
            <Button
              variant={mode === 'invite' ? 'primary' : 'outline-primary'}
              onClick={() => setMode('invite')}
            >
              Send invite email
            </Button>
            <Button
              variant={mode === 'create' ? 'primary' : 'outline-primary'}
              onClick={() => setMode('create')}
            >
              Create with password
            </Button>
          </ButtonGroup>
        </div>

        <p className="text-body-tertiary fs-10 mb-3">
          {mode === 'invite' ? (
            <>
              The user receives an email with a 7-day invite link. They pick their
              own password on the client app's <code>/accept-invite</code> page,
              and the email is marked verified on redemption.
            </>
          ) : (
            <>
              The user is created with a temp password and email already marked
              verified. Share the password securely; the user can change it once
              logged in.
            </>
          )}
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
          {mode === 'create' ? (
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
          ) : (
            <Col md={6}>
              <Form.Label className="fs-10">Inviter name (optional)</Form.Label>
              <Form.Control
                size="sm"
                placeholder="e.g. Tore at Orkestra"
                value={form.inviterName ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, inviterName: e.target.value }))}
              />
              <Form.Text className="text-body-tertiary fs-11">
                Rendered into the invite email greeting. Leave blank for a generic “You've been
                invited” opener.
              </Form.Text>
            </Col>
          )}
        </Row>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="outline-secondary" size="sm" onClick={onHide}>
          Cancel
        </Button>
        <Button variant="primary" size="sm" onClick={onSubmit} disabled={isLoading}>
          {isLoading ? (
            <Spinner size="sm" animation="border" />
          ) : mode === 'invite' ? (
            'Send invite'
          ) : (
            'Create user'
          )}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default CreateClientUserModal;
