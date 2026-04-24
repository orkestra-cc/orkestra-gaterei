import { useState } from 'react';
import { Badge, Button, Card, Spinner } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faShieldHalved } from '@fortawesome/free-solid-svg-icons';
import { useGetMfaStatusQuery } from 'store/api/mfaApi';
import MfaEnrollWizard from './MfaEnrollWizard';
import MfaRemoveModal from './MfaRemoveModal';

/**
 * Security-settings card for the current user's second factor. Renders the
 * three states the backend MFA status endpoint can return (none / pending /
 * active) and hands off to modal dialogs for the enroll + remove flows.
 */
const MfaSettings = () => {
  const { data, isLoading, refetch } = useGetMfaStatusQuery();
  const [showEnroll, setShowEnroll] = useState(false);
  const [showRemove, setShowRemove] = useState(false);

  const status = data?.status ?? 'none';
  const isActive = status === 'active';
  const isPending = status === 'pending';

  return (
    <>
      <Card className="mb-3">
        <Card.Header className="bg-body-tertiary">
          <div className="d-flex align-items-center">
            <FontAwesomeIcon icon={faShieldHalved} className="me-2 text-primary" />
            <Card.Title as="h5" className="mb-0">Two-factor authentication</Card.Title>
          </div>
        </Card.Header>
        <Card.Body>
          {isLoading ? (
            <div className="text-center py-3"><Spinner size="sm" /></div>
          ) : isActive ? (
            <div>
              <div className="d-flex align-items-center mb-2">
                <Badge bg="success" className="me-2">Enabled</Badge>
                <span className="text-muted fs-10">
                  Authenticator app (TOTP) · {data?.backupCodesRemaining ?? 0} backup codes remaining
                </span>
              </div>
              <p className="fs-10 text-muted mb-3">
                A one-time code from your authenticator is required each time you sign in.
              </p>
              <Button variant="outline-danger" size="sm" onClick={() => setShowRemove(true)}>
                Remove factor
              </Button>
            </div>
          ) : isPending ? (
            <div>
              <Badge bg="warning" className="mb-2">Enrollment in progress</Badge>
              <p className="fs-10 text-muted mb-3">
                Your authenticator is registered but never confirmed. Complete or restart enrollment below.
              </p>
              <Button variant="primary" size="sm" onClick={() => setShowEnroll(true)}>
                Resume enrollment
              </Button>
            </div>
          ) : (
            <div>
              <p className="fs-10 text-muted mb-3">
                Two-factor authentication adds a second verification step to your sign-in. You will need
                an authenticator app such as Google Authenticator, Authy, or 1Password.
              </p>
              <Button variant="primary" size="sm" onClick={() => setShowEnroll(true)}>
                Set up
              </Button>
            </div>
          )}
        </Card.Body>
      </Card>

      <MfaEnrollWizard
        show={showEnroll}
        onHide={() => { setShowEnroll(false); refetch(); }}
      />
      <MfaRemoveModal
        show={showRemove}
        onHide={() => { setShowRemove(false); refetch(); }}
      />
    </>
  );
};

export default MfaSettings;
