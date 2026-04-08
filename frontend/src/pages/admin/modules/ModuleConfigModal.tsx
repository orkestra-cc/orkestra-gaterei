import { useState } from 'react';
import { Alert, Button, Form, Modal, Spinner } from 'react-bootstrap';
import { FalconCloseButton } from 'components/common';
import type { ModuleConfig, ConfigField } from 'store/api/moduleApi';
import { useUpdateModuleMutation } from 'store/api/moduleApi';

interface ModuleConfigModalProps {
  module: ModuleConfig | null;
  show: boolean;
  onHide: () => void;
}

const ModuleConfigModal: React.FC<ModuleConfigModalProps> = ({
  module: mod,
  show,
  onHide,
}) => {
  const [updateModule, { isLoading }] = useUpdateModuleMutation();
  const [configValues, setConfigValues] = useState<Record<string, string>>({});
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [enabled, setEnabled] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Reset form when module changes
  const handleShow = () => {
    if (mod) {
      setConfigValues({ ...mod.configValues });
      setSecretValues({});
      setEnabled(mod.enabled);
      setError(null);
      setSuccess(false);
    }
  };

  const handleSave = async () => {
    if (!mod) return;
    setError(null);
    setSuccess(false);

    try {
      const body: {
        enabled?: boolean;
        config?: Record<string, string>;
        secrets?: Record<string, string>;
      } = {};

      if (enabled !== mod.enabled) {
        body.enabled = enabled;
      }

      // Collect changed config values (non-secret fields only)
      const changedConfig: Record<string, string> = {};
      for (const field of mod.configSchema) {
        if (field.type === 'secret') continue;
        if (configValues[field.key] !== mod.configValues[field.key]) {
          changedConfig[field.key] = configValues[field.key] || '';
        }
      }
      if (Object.keys(changedConfig).length > 0) {
        body.config = changedConfig;
      }

      // Collect non-empty secret values
      const newSecrets: Record<string, string> = {};
      for (const [key, value] of Object.entries(secretValues)) {
        if (value.trim()) {
          newSecrets[key] = value;
        }
      }
      if (Object.keys(newSecrets).length > 0) {
        body.secrets = newSecrets;
      }

      if (Object.keys(body).length === 0) {
        onHide();
        return;
      }

      await updateModule({ name: mod.moduleName, ...body }).unwrap();
      setSuccess(true);
      setTimeout(onHide, 800);
    } catch (err: unknown) {
      const message =
        err && typeof err === 'object' && 'data' in err
          ? String((err as { data: { detail?: string } }).data?.detail || 'Update failed')
          : 'Update failed';
      setError(message);
    }
  };

  const renderField = (field: ConfigField) => {
    const key = field.key;

    if (field.type === 'secret') {
      return (
        <Form.Group key={key} className="mb-3">
          <Form.Label className="fs-10 fw-semibold">
            {field.label}
            {mod?.secretStatus[key] && (
              <span className="badge badge-subtle-success ms-2 fs-11">Set</span>
            )}
          </Form.Label>
          <Form.Control
            type="password"
            size="sm"
            placeholder={mod?.secretStatus[key] ? 'Leave empty to keep current' : 'Enter value'}
            value={secretValues[key] || ''}
            onChange={(e) =>
              setSecretValues((prev) => ({ ...prev, [key]: e.target.value }))
            }
          />
          {field.description && (
            <Form.Text className="text-muted">{field.description}</Form.Text>
          )}
        </Form.Group>
      );
    }

    if (field.type === 'bool') {
      return (
        <Form.Group key={key} className="mb-3">
          <Form.Check
            type="switch"
            label={field.label}
            checked={configValues[key] === 'true'}
            onChange={(e) =>
              setConfigValues((prev) => ({
                ...prev,
                [key]: e.target.checked ? 'true' : 'false',
              }))
            }
          />
          {field.description && (
            <Form.Text className="text-muted">{field.description}</Form.Text>
          )}
        </Form.Group>
      );
    }

    return (
      <Form.Group key={key} className="mb-3">
        <Form.Label className="fs-10 fw-semibold">{field.label}</Form.Label>
        <Form.Control
          type={field.type === 'int' ? 'number' : 'text'}
          size="sm"
          placeholder={field.default || ''}
          value={configValues[key] || ''}
          onChange={(e) =>
            setConfigValues((prev) => ({ ...prev, [key]: e.target.value }))
          }
        />
        {field.envVar && (
          <Form.Text className="text-muted">
            Env: <code>{field.envVar}</code>
            {field.description ? ` — ${field.description}` : ''}
          </Form.Text>
        )}
      </Form.Group>
    );
  };

  if (!mod) return null;

  const isCore = mod.category === 'core';
  const hasSchema = mod.configSchema && mod.configSchema.length > 0;

  return (
    <Modal show={show} onHide={onHide} onShow={handleShow} size="lg">
      <Modal.Header>
        <Modal.Title className="fs-8">
          {mod.displayName}
          <span className="text-muted fs-10 ms-2">({mod.moduleName})</span>
        </Modal.Title>
        <FalconCloseButton onClick={onHide} />
      </Modal.Header>
      <Modal.Body>
        {error && <Alert variant="danger" className="fs-10">{error}</Alert>}
        {success && <Alert variant="success" className="fs-10">Module updated successfully</Alert>}

        <Form.Group className="mb-4">
          <Form.Check
            type="switch"
            id="module-enabled"
            label={enabled ? 'Enabled' : 'Disabled'}
            checked={enabled}
            disabled={isCore}
            onChange={(e) => setEnabled(e.target.checked)}
          />
          {isCore && (
            <Form.Text className="text-muted">Core modules cannot be disabled</Form.Text>
          )}
        </Form.Group>

        {mod.status === 'failed' && mod.error && (
          <Alert variant="danger" className="fs-10">
            <strong>Init Error:</strong> {mod.error}
          </Alert>
        )}

        {hasSchema && (
          <>
            <h6 className="fs-9 border-bottom pb-2 mb-3">Configuration</h6>
            {mod.configSchema.map(renderField)}
          </>
        )}

        {mod.dependsOn && mod.dependsOn.length > 0 && (
          <div className="mt-3 fs-10 text-muted">
            <strong>Dependencies:</strong> {mod.dependsOn.join(', ')}
          </div>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" size="sm" onClick={onHide}>
          Cancel
        </Button>
        <Button variant="primary" size="sm" onClick={handleSave} disabled={isLoading}>
          {isLoading ? <Spinner animation="border" size="sm" /> : 'Save Changes'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default ModuleConfigModal;
