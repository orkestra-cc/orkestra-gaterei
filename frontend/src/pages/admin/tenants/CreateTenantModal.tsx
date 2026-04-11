import { useEffect, useState } from 'react';
import { Button, Form, Modal, Spinner } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { toast } from 'react-toastify';
import { useCreateOrgMutation } from 'store/api/tenantApi';
import { baseApi } from 'store/api/baseApi';
import { useAppDispatch } from 'store/hooks';

interface Props {
  show: boolean;
  onHide: () => void;
}

function slugify(input: string): string {
  return input
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s_-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
}

const CreateTenantModal: React.FC<Props> = ({ show, onHide }) => {
  const dispatch = useAppDispatch();
  const [createOrg, { isLoading }] = useCreateOrgMutation();

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [slugEdited, setSlugEdited] = useState(false);
  const [plan, setPlan] = useState('free');

  useEffect(() => {
    if (!show) {
      setName('');
      setSlug('');
      setSlugEdited(false);
      setPlan('free');
    }
  }, [show]);

  const handleNameChange = (value: string) => {
    setName(value);
    if (!slugEdited) {
      setSlug(slugify(value));
    }
  };

  const canSave = name.trim().length > 0 && slug.trim().length > 0 && !isLoading;

  const onSave = async () => {
    try {
      await createOrg({ name: name.trim(), slug: slug.trim(), plan }).unwrap();
      toast.success(`Tenant "${name.trim()}" created`);
      // Refresh the platform-admin list so the new org appears immediately.
      dispatch(baseApi.util.invalidateTags([{ type: 'AdminOrg', id: 'LIST' }]));
      onHide();
    } catch (err: unknown) {
      toast.error('Create failed: ' + extractError(err));
    }
  };

  return (
    <Modal show={show} onHide={onHide} backdrop="static" centered>
      <Modal.Header closeButton>
        <Modal.Title>
          <FontAwesomeIcon icon="plus" className="text-primary me-2" />
          Create tenant
        </Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <Form>
          <Form.Group className="mb-3">
            <Form.Label className="fw-semibold">Name</Form.Label>
            <Form.Control
              type="text"
              placeholder="e.g. Acme Corp"
              value={name}
              maxLength={120}
              onChange={(e) => handleNameChange(e.target.value)}
              autoFocus
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label className="fw-semibold">Slug</Form.Label>
            <Form.Control
              type="text"
              placeholder="acme-corp"
              value={slug}
              maxLength={80}
              onChange={(e) => {
                setSlug(slugify(e.target.value));
                setSlugEdited(true);
              }}
            />
            <Form.Text muted>
              Lowercase letters, numbers and dashes. Auto-generated from the
              name unless you override it.
            </Form.Text>
          </Form.Group>
          <Form.Group className="mb-0">
            <Form.Label className="fw-semibold">Plan</Form.Label>
            <Form.Select value={plan} onChange={(e) => setPlan(e.target.value)}>
              <option value="free">Free</option>
              <option value="pro">Pro</option>
              <option value="enterprise">Enterprise</option>
            </Form.Select>
          </Form.Group>
        </Form>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={onHide} disabled={isLoading}>
          Cancel
        </Button>
        <Button variant="primary" onClick={onSave} disabled={!canSave}>
          {isLoading ? (
            <>
              <Spinner size="sm" animation="border" className="me-2" /> Creating…
            </>
          ) : (
            <>Create tenant</>
          )}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

function extractError(err: unknown): string {
  if (err && typeof err === 'object' && 'data' in err) {
    const data = (err as { data?: { detail?: string; title?: string } }).data;
    return data?.detail || data?.title || 'unknown error';
  }
  return String(err);
}

export default CreateTenantModal;
