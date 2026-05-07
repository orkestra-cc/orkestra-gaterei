import { useState } from 'react';
import { Alert, Button, Form, Modal, Spinner } from 'react-bootstrap';
import { useNavigate } from 'react-router';
import {
  useDeleteClientUserAdminMutation,
  type AdminClientUserItem,
} from 'store/api/userApi';

interface Props {
  show: boolean;
  onHide: () => void;
  user: AdminClientUserItem;
}

// Two-step confirmation modal — the admin must type the user's email
// before the destructive action enables, mirroring the pattern used on
// the operator user-management page. Calls SoftDeleteAndAliasEmail
// server-side so the address is freed for a fresh signup.
const DeleteClientUserModal: React.FC<Props> = ({ show, onHide, user }) => {
  const navigate = useNavigate();
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [deleteUser, { isLoading }] = useDeleteClientUserAdminMutation();

  const matches = confirm.trim().toLowerCase() === user.email.toLowerCase();

  const onSubmit = async () => {
    setError(null);
    try {
      await deleteUser(user.id).unwrap();
      onHide();
      navigate('/admin/clients');
    } catch (err) {
      const msg =
        (err as { data?: { detail?: string; message?: string } }).data?.detail ??
        (err as { data?: { detail?: string; message?: string } }).data?.message ??
        'Failed to delete user.';
      setError(msg);
    }
  };

  return (
    <Modal show={show} onHide={onHide} centered>
      <Modal.Header closeButton>
        <Modal.Title>Delete client user</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {error && <Alert variant="danger">{error}</Alert>}
        <p>
          This will soft-delete <strong>{user.email}</strong> and rewrite their email to
          a one-shot alias so the address can be reused for a new signup. Tenant
          memberships remain in the database for audit but the user can no longer log
          in. This action cannot be undone from the UI.
        </p>
        <Form.Group>
          <Form.Label className="fs-10">
            Type the user's email <code>{user.email}</code> to confirm:
          </Form.Label>
          <Form.Control
            size="sm"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder={user.email}
          />
        </Form.Group>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="outline-secondary" size="sm" onClick={onHide}>
          Cancel
        </Button>
        <Button
          variant="danger"
          size="sm"
          onClick={onSubmit}
          disabled={!matches || isLoading}
        >
          {isLoading ? <Spinner size="sm" animation="border" /> : 'Delete user'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default DeleteClientUserModal;
