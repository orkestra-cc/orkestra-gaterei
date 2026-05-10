import { useState } from 'react';
import { Alert, Badge, Button, Card, Modal, Spinner, Table } from 'react-bootstrap';
import {
  useGetSelfAuthMethodsQuery,
  useUnlinkOauthSelfMutation,
  type OAuthProvider,
  type SelfAuthOAuthProvider,
} from 'store/api/authApi';

const PROVIDER_LABELS: Record<OAuthProvider, string> = {
  google: 'Google',
  apple: 'Apple',
  github: 'GitHub',
  discord: 'Discord',
};

// LinkedProvidersTab lists the OAuth identities the user has linked
// and exposes a per-row Unlink action. The unlink endpoint is gated
// server-side by RequireStepUp(5m); the global StepUpModal pauses
// the request, drives the user through /mfa/verify, and replays.
const LinkedProvidersTab = () => {
  const { data, isLoading, isFetching } = useGetSelfAuthMethodsQuery();
  const [unlink, { isLoading: unlinkPending }] = useUnlinkOauthSelfMutation();
  const [target, setTarget] = useState<SelfAuthOAuthProvider | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="text-center py-4">
        <Spinner animation="border" size="sm" />
      </div>
    );
  }

  const providers = data?.oauthProviders ?? [];
  const onlyCredential =
    !data?.hasUsablePassword && providers.length === 1;

  const onConfirmUnlink = async () => {
    if (!target) return;
    setError(null);
    try {
      await unlink({ provider: target.provider }).unwrap();
      setTarget(null);
    } catch (err: unknown) {
      const e = err as { data?: { detail?: string; title?: string; code?: string } };
      const code = e?.data?.code;
      if (code === 'last_credential') {
        setError(
          'You have no other login method. Set a password before unlinking this provider.',
        );
      } else if (code === 'step_up_required') {
        // The global StepUpModal will pick this up and replay; close
        // the inline modal so the prompt isn't obscured.
        setTarget(null);
      } else {
        setError(e?.data?.detail || e?.data?.title || 'Failed to unlink provider.');
      }
    }
  };

  return (
    <>
      <Card className="shadow-none border">
        <Card.Header>
          <Card.Title as="h5" className="mb-0">
            Linked sign-in providers
          </Card.Title>
        </Card.Header>
        <Card.Body>
          {providers.length === 0 ? (
            <p className="fs-10 text-muted mb-0">
              No sign-in providers linked. Link one from the login page to
              add a second authentication method.
            </p>
          ) : (
            <>
              {onlyCredential && (
                <Alert variant="warning" className="fs-10">
                  This is your only login method. Unlinking it would lock you
                  out — set a password first.
                </Alert>
              )}
              <Table responsive size="sm" className="mb-0 align-middle">
                <thead>
                  <tr>
                    <th>Provider</th>
                    <th>Email</th>
                    <th>Linked</th>
                    <th className="text-end">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {providers.map((p) => (
                    <tr key={p.provider}>
                      <td>
                        {PROVIDER_LABELS[p.provider]}
                        {p.isPrimary && (
                          <Badge bg="primary" className="ms-2">
                            primary
                          </Badge>
                        )}
                      </td>
                      <td className="fs-10">{p.email}</td>
                      <td className="fs-10 text-muted">
                        {new Date(p.linkedAt).toLocaleDateString()}
                      </td>
                      <td className="text-end">
                        <Button
                          variant="outline-danger"
                          size="sm"
                          disabled={onlyCredential || isFetching}
                          onClick={() => setTarget(p)}
                        >
                          Unlink
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            </>
          )}
        </Card.Body>
      </Card>

      <Modal show={!!target} onHide={() => setTarget(null)} centered>
        <Modal.Header closeButton>
          <Modal.Title>Unlink {target ? PROVIDER_LABELS[target.provider] : ''}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {error && (
            <Alert variant="danger" className="fs-10">
              {error}
            </Alert>
          )}
          <p className="mb-0">
            Remove the {target ? PROVIDER_LABELS[target.provider] : ''} sign-in
            method? You can re-link it later from the login screen.
          </p>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setTarget(null)}>
            Cancel
          </Button>
          <Button variant="danger" onClick={onConfirmUnlink} disabled={unlinkPending}>
            {unlinkPending ? 'Unlinking…' : 'Unlink'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default LinkedProvidersTab;
