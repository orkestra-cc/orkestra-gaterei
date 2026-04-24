import { useEffect, useState, FormEvent } from 'react';
import { Alert, Button, Form, Modal } from 'react-bootstrap';
import { useVerifyMfaMutation } from 'store/api/mfaApi';
import { subscribeStepUp, completeStepUp } from 'store/stepUp';

/**
 * Global step-up verification modal. Opens whenever the RTK Query base
 * query calls requestStepUp() in response to a 401 with
 * code="step_up_required". Collects a TOTP (or backup) code, POSTs
 * /v1/auth/mfa/verify — the mutation dispatches the refreshed access
 * token into Redux — and then signals the paused base-query callers via
 * completeStepUp(true) so they replay the original request.
 *
 * Cancel resolves waiters with false, letting the original 401 surface
 * to the component that issued the request.
 */
const StepUpModal = () => {
  const [open, setOpen] = useState(false);
  const [code, setCode] = useState('');
  const [useBackup, setUseBackup] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [verify, { isLoading }] = useVerifyMfaMutation();

  useEffect(() => {
    return subscribeStepUp((next) => {
      setOpen(next);
      if (!next) {
        setCode('');
        setUseBackup(false);
        setError(null);
      }
    });
  }, []);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    if (!code.trim()) {
      setError('Enter the code from your authenticator.');
      return;
    }
    try {
      await verify({ code: code.trim(), useBackup }).unwrap();
      // Fresh access token is already in Redux thanks to verifyMfa's
      // onQueryStarted — signal the paused requests to replay.
      completeStepUp(true);
    } catch (err: unknown) {
      const anyErr = err as { status?: number; data?: { detail?: string } };
      if (anyErr?.status === 401) {
        setError('Incorrect code. Please try again.');
      } else if (anyErr?.status === 429) {
        setError('Too many attempts. Please wait a moment and try again.');
      } else {
        setError(anyErr?.data?.detail ?? 'Could not verify the code. Please try again.');
      }
    }
  };

  const handleCancel = () => {
    if (isLoading) return;
    completeStepUp(false);
  };

  return (
    <Modal show={open} onHide={handleCancel} backdrop="static" centered>
      <Modal.Header closeButton={!isLoading}>
        <Modal.Title>Confirm this action</Modal.Title>
      </Modal.Header>
      <Form onSubmit={handleSubmit} noValidate>
        <Modal.Body>
          <p className="fs-10 mb-3">
            For your security, enter a fresh code from your authenticator to continue.
          </p>

          {error && (
            <Alert variant="danger" className="mb-3" onClose={() => setError(null)} dismissible>
              {error}
            </Alert>
          )}

          <Form.Group className="mb-2">
            <Form.Label>{useBackup ? 'Backup code' : 'Authenticator code'}</Form.Label>
            <Form.Control
              type="text"
              inputMode={useBackup ? 'text' : 'numeric'}
              autoComplete="one-time-code"
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder={useBackup ? 'XXXX-XXXX' : '123 456'}
              required
            />
          </Form.Group>
          <button
            type="button"
            className="btn btn-link p-0 fs-10"
            onClick={() => { setUseBackup((v) => !v); setCode(''); }}
          >
            {useBackup ? 'Use authenticator app instead' : 'Use a backup code instead'}
          </button>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="outline-secondary" onClick={handleCancel} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={isLoading}>
            {isLoading ? 'Verifying…' : 'Verify and continue'}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default StepUpModal;
