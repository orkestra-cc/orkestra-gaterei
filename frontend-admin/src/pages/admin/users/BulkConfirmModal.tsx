import { FormEvent, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Form, ListGroup, Modal } from 'react-bootstrap';
import { Trans, useTranslation } from 'react-i18next';
import { toast } from 'react-toastify';
import {
  useDeleteUserMutation,
  useUpdateUserMutation,
  User
} from 'store/api/userApi';

export type BulkAction = 'activate' | 'deactivate' | 'delete';

interface Props {
  show: boolean;
  action: BulkAction | null;
  // The full selection coming from the table — the modal filters out the
  // current user itself so the operation never tries to act on the caller.
  selectedUsers: User[];
  currentUserId?: string;
  onHide: () => void;
  // Called after the operation finishes (success, partial, or all-failed)
  // so the parent can clear table selection and refetch the list. Not
  // invoked on plain "cancel before run" closes — the parent already owns
  // the modal's `show` toggle for that.
  onCompleted?: () => void;
}

interface Failure {
  user: User;
  reason: string;
}

const PREVIEW_LIMIT = 10;

// extractMessage prefers errcode `code` (translated), then `detail`, then
// a generic label — same shape as the row-action helper.
function extractMessage(
  err: unknown,
  t: (key: string) => string,
  fallback: string
): string {
  if (err && typeof err === 'object' && 'data' in err) {
    const data = (err as { data?: { code?: string; detail?: string } }).data;
    if (data?.code) {
      const translated = t(`errors.${data.code}`);
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

const BulkConfirmModal: React.FC<Props> = ({
  show,
  action,
  selectedUsers,
  currentUserId,
  onHide,
  onCompleted
}) => {
  const { t } = useTranslation();

  const [updateUser] = useUpdateUserMutation();
  const [deleteUser] = useDeleteUserMutation();

  const [confirmText, setConfirmText] = useState('');
  const [running, setRunning] = useState(false);
  const [doneCount, setDoneCount] = useState(0);
  const [failures, setFailures] = useState<Failure[]>([]);
  const [completed, setCompleted] = useState(false);

  // Reset every time the modal opens so a previous run never leaks into
  // the next one. `action` is part of the key because re-opening for a
  // different bulk action should start fresh too.
  useEffect(() => {
    if (show) {
      setConfirmText('');
      setRunning(false);
      setDoneCount(0);
      setFailures([]);
      setCompleted(false);
    }
  }, [show, action]);

  // Self-exclusion + de-duplication. The table can in theory feed the
  // same user twice across page transitions; we guard defensively here.
  const eligibleUsers = useMemo(() => {
    const seen = new Set<string>();
    return selectedUsers.filter(u => {
      if (u.id === currentUserId) return false;
      if (seen.has(u.id)) return false;
      seen.add(u.id);
      return true;
    });
  }, [selectedUsers, currentUserId]);

  const selfExcluded = useMemo(
    () => selectedUsers.find(u => u.id === currentUserId) ?? null,
    [selectedUsers, currentUserId]
  );

  if (!action) return null;

  const total = eligibleUsers.length;

  const title = t(
    action === 'activate'
      ? 'adminUsers.bulkModal.titleActivate'
      : action === 'deactivate'
        ? 'adminUsers.bulkModal.titleDeactivate'
        : 'adminUsers.bulkModal.titleDelete',
    { count: total }
  );
  const runLabel = t(
    action === 'activate'
      ? 'adminUsers.bulkModal.runActivate'
      : action === 'deactivate'
        ? 'adminUsers.bulkModal.runDeactivate'
        : 'adminUsers.bulkModal.runDelete'
  );
  const runVariant = action === 'delete' ? 'danger' : 'warning';

  const requiresTypedConfirm = action === 'delete';
  const typedConfirmOK =
    !requiresTypedConfirm || confirmText.trim() === 'DELETE';
  const canSubmit = total > 0 && typedConfirmOK && !running && !completed;

  const actionLabel = (a: BulkAction) =>
    t(`adminUsers.bulkModal.actionLabel.${a}`);

  const runOne = async (user: User): Promise<void> => {
    if (action === 'delete') {
      await deleteUser(user.id).unwrap();
      return;
    }
    await updateUser({
      id: user.id,
      data: { isActive: action === 'activate' }
    }).unwrap();
  };

  const handleRun = async (event: FormEvent) => {
    event.preventDefault();
    if (!canSubmit) return;
    setRunning(true);
    setDoneCount(0);
    setFailures([]);

    const fallback = t('adminUsers.mfaReset.errors.generic');
    const localFailures: Failure[] = [];

    // Sequential is cheap (small N), keeps the progress counter honest,
    // and avoids hammering the backend when 50 rows are selected.
    for (const user of eligibleUsers) {
      try {
        await runOne(user);
      } catch (err) {
        localFailures.push({
          user,
          reason: extractMessage(err, t, fallback)
        });
      } finally {
        setDoneCount(d => d + 1);
      }
    }

    setFailures(localFailures);
    setCompleted(true);
    setRunning(false);

    const ok = eligibleUsers.length - localFailures.length;
    if (localFailures.length === 0) {
      toast.success(
        t('adminUsers.bulkModal.summarySuccess', {
          count: eligibleUsers.length
        })
      );
    } else if (ok === 0) {
      toast.error(
        t('adminUsers.bulkModal.summaryAllFailed', {
          count: eligibleUsers.length
        })
      );
    } else {
      toast.warning(
        t('adminUsers.bulkModal.summaryPartial', {
          ok,
          failed: localFailures.length,
          action: actionLabel(action)
        })
      );
    }

    onCompleted?.();
  };

  const handleClose = () => {
    if (running) return;
    onHide();
  };

  const previewList = eligibleUsers.slice(0, PREVIEW_LIMIT);
  const truncatedCount = eligibleUsers.length - previewList.length;

  return (
    <Modal show={show} onHide={handleClose} centered>
      <Modal.Header closeButton={!running}>
        <Modal.Title>{title}</Modal.Title>
      </Modal.Header>
      <Form onSubmit={handleRun} noValidate>
        <Modal.Body>
          <p className="mb-2">
            {t('adminUsers.bulkModal.intro', { count: eligibleUsers.length })}
          </p>

          {selfExcluded && (
            <Alert variant="info" className="mb-3">
              {t('adminUsers.bulkModal.selfSkipped', {
                email: selfExcluded.email
              })}
            </Alert>
          )}

          {action === 'delete' && (
            <Alert variant="warning" className="mb-3">
              {t('adminUsers.bulkModal.warningDelete')}
            </Alert>
          )}

          {total === 0 ? (
            <Alert variant="secondary" className="mb-3">
              {t('adminUsers.bulkModal.noEligible')}
            </Alert>
          ) : (
            <>
              <p className="fw-semibold mb-2">
                {t('adminUsers.bulkModal.listHeading')}
              </p>
              <ListGroup variant="flush" className="mb-3">
                {previewList.map(u => (
                  <ListGroup.Item key={u.id} className="px-0 py-1 fs-10">
                    <span className="fw-semibold">{u.fullName || u.email}</span>{' '}
                    <span className="text-muted">{u.email}</span>
                  </ListGroup.Item>
                ))}
                {truncatedCount > 0 && (
                  <ListGroup.Item className="px-0 py-1 fs-10 text-muted">
                    {t('adminUsers.bulkModal.listTruncated', {
                      count: truncatedCount
                    })}
                  </ListGroup.Item>
                )}
              </ListGroup>
            </>
          )}

          {requiresTypedConfirm && total > 0 && !completed && (
            <Form.Group>
              <Form.Label>
                <Trans
                  i18nKey="adminUsers.bulkModal.confirmPrompt"
                  components={{ strong: <strong /> }}
                />
              </Form.Label>
              <Form.Control
                type="text"
                value={confirmText}
                onChange={e => setConfirmText(e.target.value)}
                placeholder={t('adminUsers.bulkModal.confirmPlaceholder')}
                autoComplete="off"
                spellCheck={false}
                disabled={running}
              />
            </Form.Group>
          )}

          {(running || completed) && (
            <div className="mt-3 fs-10 text-muted" data-testid="bulk-progress">
              {t('adminUsers.bulkModal.progress', {
                done: doneCount,
                total: eligibleUsers.length
              })}
            </div>
          )}

          {completed && failures.length > 0 && (
            <Alert variant="danger" className="mt-3 mb-0">
              <ul className="mb-0 ps-3">
                {failures.map(f => (
                  <li key={f.user.id}>
                    <span className="fw-semibold">
                      {f.user.fullName || f.user.email}
                    </span>
                    : {f.reason}
                  </li>
                ))}
              </ul>
            </Alert>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="outline-secondary"
            onClick={handleClose}
            disabled={running}
          >
            {completed
              ? t('adminUsers.bulkModal.close')
              : t('adminUsers.bulkModal.cancel')}
          </Button>
          {!completed && (
            <Button type="submit" variant={runVariant} disabled={!canSubmit}>
              {running ? t('adminUsers.bulkModal.running') : runLabel}
            </Button>
          )}
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default BulkConfirmModal;
