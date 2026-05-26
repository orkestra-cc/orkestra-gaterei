import { FormEvent, useEffect, useState } from 'react';
import { Alert, Button, Form, Modal } from 'react-bootstrap';
import { Trans, useTranslation } from 'react-i18next';
import { toast } from 'react-toastify';
import { useDeleteUserMutation, User } from 'store/api/userApi';

interface Props {
  show: boolean;
  user: User | null;
  onHide: () => void;
}

// extractErrorMessage prefers the typed errcode `code` returned by the
// backend (translated via the `errors.<code>` namespace), falling back to
// the human-readable `detail` and finally a generic label. Lives inline
// because we do not want a shared helper to absorb fields none of the
// other call sites need (e.g. status).
function extractErrorMessage(
  err: unknown,
  t: (key: string) => string,
  fallback: string
): string {
  if (err && typeof err === 'object' && 'data' in err) {
    const data = (err as { data?: { code?: string; detail?: string } }).data;
    if (data?.code) {
      const translated = t(`errors.${data.code}`);
      // i18next returns the key itself on a miss, so only adopt the
      // translation when it actually resolved to something else.
      if (translated && translated !== `errors.${data.code}`) {
        return translated;
      }
    }
    if (data?.detail) {
      return data.detail;
    }
  }
  return fallback;
}

const DeleteUserModal: React.FC<Props> = ({ show, user, onHide }) => {
  const { t } = useTranslation();
  const [confirmText, setConfirmText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [deleteUser, { isLoading }] = useDeleteUserMutation();

  // Reset local state every time a new user is targeted so a previous
  // failed attempt's input never leaks into the next confirmation.
  useEffect(() => {
    if (show) {
      setConfirmText('');
      setError(null);
    }
  }, [show, user?.id]);

  if (!user) return null;

  const canSubmit = confirmText.trim() === user.email && !isLoading;

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!canSubmit) return;
    setError(null);
    try {
      await deleteUser(user.id).unwrap();
      toast.success(
        t('adminUsers.rowActions.toastDeleted', {
          user: user.fullName || user.email
        })
      );
      onHide();
    } catch (err) {
      const message = extractErrorMessage(
        err,
        t,
        t('adminUsers.rowActions.toastDeleteFailed', {
          user: user.fullName || user.email,
          error: ''
        })
      );
      setError(message);
    }
  };

  const handleClose = () => {
    if (isLoading) return;
    onHide();
  };

  return (
    <Modal show={show} onHide={handleClose} centered>
      <Modal.Header closeButton={!isLoading}>
        <Modal.Title>{t('adminUsers.deleteModal.title')}</Modal.Title>
      </Modal.Header>
      <Form onSubmit={handleSubmit} noValidate>
        <Modal.Body>
          <p className="mb-2">
            <Trans
              i18nKey="adminUsers.deleteModal.intro"
              values={{ user: user.fullName || user.email }}
              components={{ strong: <strong /> }}
            />
          </p>
          <Alert variant="warning" className="mb-3">
            <Trans
              i18nKey="adminUsers.deleteModal.warning"
              values={{ email: user.email }}
              components={{ strong: <strong /> }}
            />
          </Alert>

          {error && (
            <Alert
              variant="danger"
              className="mb-3"
              onClose={() => setError(null)}
              dismissible
            >
              {error}
            </Alert>
          )}

          <Form.Group>
            <Form.Label>{t('adminUsers.deleteModal.confirmPrompt')}</Form.Label>
            <Form.Control
              type="text"
              autoFocus
              value={confirmText}
              onChange={e => setConfirmText(e.target.value)}
              placeholder={t('adminUsers.deleteModal.confirmPlaceholder')}
              autoComplete="off"
              spellCheck={false}
              required
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="outline-secondary"
            onClick={handleClose}
            disabled={isLoading}
          >
            {t('adminUsers.deleteModal.cancel')}
          </Button>
          <Button type="submit" variant="danger" disabled={!canSubmit}>
            {isLoading
              ? t('adminUsers.deleteModal.submitting')
              : t('adminUsers.deleteModal.submit')}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default DeleteUserModal;
