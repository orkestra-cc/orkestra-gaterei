import { useState, useEffect } from 'react';
import { Row, Col, Card, Form, Button, Spinner, Alert } from 'react-bootstrap';
import { Link } from 'react-router-dom';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCog, faSave, faExternalLinkAlt } from '@fortawesome/free-solid-svg-icons';
import Background from 'components/common/Background';
import greetingsBg from 'assets/img/illustrations/ticket-greetings-bg.png';
import { useGetSalesSettingsQuery, useUpdateSalesSettingsMutation } from '../../../store/api/salesApi';
import { useListAIModelsQuery } from '../../../store/api/aiModelsApi';

const SalesSettingsPage = () => {
  const { data: settings, isLoading: settingsLoading } = useGetSalesSettingsQuery();
  const { data: modelsData, isLoading: modelsLoading } = useListAIModelsQuery({ type: 'llm' });
  const [updateSettings, { isLoading: saving }] = useUpdateSalesSettingsMutation();

  const [modelUuid, setModelUuid] = useState('');
  const [temperature, setTemperature] = useState(0);
  const [maxTokens, setMaxTokens] = useState(0);
  const [locale, setLocale] = useState('');
  const [saved, setSaved] = useState(false);

  const models = modelsData?.models || [];

  // Sync form from server settings
  useEffect(() => {
    if (settings) {
      setModelUuid(settings.modelUuid || '');
      setTemperature(settings.temperature || 0);
      setMaxTokens(settings.maxTokens || 0);
      setLocale(settings.locale || '');
    }
  }, [settings]);

  const handleSave = async () => {
    setSaved(false);
    await updateSettings({ modelUuid, temperature, maxTokens, locale }).unwrap();
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  const selectedModel = models.find((m: any) => m.uuid === modelUuid);
  const defaultModel = models.find((m: any) => m.isDefault);

  if (settingsLoading || modelsLoading) {
    return <div className="text-center py-5"><Spinner /></div>;
  }

  return (
    <>
      <Row className="g-3 mb-3">
        <Col xxl={12}>
          <Card className="bg-body-tertiary dark__bg-opacity-50 shadow-none h-100">
            <Background image={greetingsBg} className="bg-card d-none d-sm-block" />
            <Card.Header className="d-flex align-items-center z-1 p-0">
              <div className="bg-primary rounded-circle p-3 ms-3">
                <FontAwesomeIcon icon={faCog} className="text-white" size="2x" />
              </div>
              <div className="ms-3">
                <h6 className="mb-1 text-primary">Sales Intelligence</h6>
                <h4 className="mb-0 text-primary fw-bold">
                  Settings
                </h4>
              </div>
            </Card.Header>
          </Card>
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        <Col lg={8}>
          <Card>
            <Card.Header><h5 className="mb-0">LLM Configuration</h5></Card.Header>
            <Card.Body>
              <Form>
                {/* Model Selection */}
                <Form.Group className="mb-4">
                  <Form.Label className="fw-semibold">AI Model</Form.Label>
                  <Form.Select value={modelUuid} onChange={e => setModelUuid(e.target.value)}>
                    <option value="">
                      System Default{defaultModel ? ` (${defaultModel.name} — ${defaultModel.modelName})` : ''}
                    </option>
                    {models.map((m: any) => (
                      <option key={m.uuid} value={m.uuid}>
                        {m.name} — {m.modelName}
                        {m.provider !== 'ollama' ? ` (${m.provider})` : ''}
                        {m.isDefault ? ' ★' : ''}
                        {!m.isActive ? ' [disabled]' : ''}
                      </option>
                    ))}
                  </Form.Select>
                  <Form.Text className="text-muted">
                    Select which LLM to use for prospect analysis and skills.{' '}
                    <Link to="/ai/models">
                      Manage models <FontAwesomeIcon icon={faExternalLinkAlt} size="xs" />
                    </Link>
                  </Form.Text>
                </Form.Group>

                {/* Temperature */}
                <Form.Group className="mb-4">
                  <Form.Label className="fw-semibold">
                    Temperature: {temperature > 0 ? temperature.toFixed(2) : 'Model default'}
                  </Form.Label>
                  <Form.Range
                    min={0}
                    max={1}
                    step={0.05}
                    value={temperature}
                    onChange={e => setTemperature(parseFloat(e.target.value))}
                  />
                  <Form.Text className="text-muted">
                    0 = use model default. Lower = more focused/deterministic. Higher = more creative.
                  </Form.Text>
                </Form.Group>

                {/* Max Tokens */}
                <Form.Group className="mb-4">
                  <Form.Label className="fw-semibold">Max Output Tokens</Form.Label>
                  <Form.Control
                    type="number"
                    min={0}
                    max={128000}
                    step={256}
                    value={maxTokens}
                    onChange={e => setMaxTokens(parseInt(e.target.value) || 0)}
                    placeholder="0 = model default (4096)"
                  />
                  <Form.Text className="text-muted">
                    Maximum tokens in LLM responses. 0 uses the default (4096).
                  </Form.Text>
                </Form.Group>

                {/* Locale */}
                <Form.Group className="mb-4">
                  <Form.Label className="fw-semibold">Default Locale</Form.Label>
                  <Form.Select value={locale} onChange={e => setLocale(e.target.value)}>
                    <option value="">System default (Italian)</option>
                    <option value="it">Italian</option>
                    <option value="en">English</option>
                  </Form.Select>
                  <Form.Text className="text-muted">
                    Controls prompt language and cultural adaptation for outreach, proposals, and qualification.
                  </Form.Text>
                </Form.Group>

                <div className="d-flex align-items-center gap-3">
                  <Button variant="primary" onClick={handleSave} disabled={saving}>
                    {saving ? <Spinner size="sm" className="me-1" /> : <FontAwesomeIcon icon={faSave} className="me-1" />}
                    Save Settings
                  </Button>
                  {saved && <Alert variant="success" className="mb-0 py-1 px-3">Saved!</Alert>}
                </div>
              </Form>
            </Card.Body>
          </Card>
        </Col>

        <Col lg={4}>
          <Card>
            <Card.Header><h5 className="mb-0">Active Configuration</h5></Card.Header>
            <Card.Body>
              <dl className="mb-0">
                <dt>Model</dt>
                <dd className="text-muted">
                  {selectedModel
                    ? `${selectedModel.name} (${selectedModel.modelName})`
                    : defaultModel
                      ? `${defaultModel.name} (${defaultModel.modelName}) — system default`
                      : 'No LLM configured'}
                </dd>

                <dt>Provider</dt>
                <dd className="text-muted">
                  {(selectedModel || defaultModel)?.provider || '—'}
                  {(selectedModel || defaultModel)?.providerCategory === 'local' ? ' (local)' : ' (cloud)'}
                </dd>

                <dt>Temperature</dt>
                <dd className="text-muted">
                  {temperature > 0 ? temperature.toFixed(2) : `Model default (${(selectedModel || defaultModel)?.temperature || 0.1})`}
                </dd>

                <dt>Max Tokens</dt>
                <dd className="text-muted">
                  {maxTokens > 0 ? maxTokens : `Default (${(selectedModel || defaultModel)?.maxTokens || 4096})`}
                </dd>

                <dt>Locale</dt>
                <dd className="text-muted">{locale || 'Italian (default)'}</dd>
              </dl>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </>
  );
};

export default SalesSettingsPage;
