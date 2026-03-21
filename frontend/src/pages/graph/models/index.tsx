import { useState, useCallback } from 'react';
import { Row, Col, Card, Button, Form, Table, Badge, Spinner, Alert, Modal } from 'react-bootstrap';
import {
  useListModelsQuery,
  useCreateModelMutation,
  useUpdateModelMutation,
  useDeleteModelMutation,
  useSetDefaultModelMutation,
  useTestModelMutation,
  useFetchProviderModelsMutation,
} from '../../../store/api/ragApi';
import type { ModelConfig, CreateModelRequest, UpdateModelRequest } from '../../../types/rag';

const emptyForm: CreateModelRequest = {
  name: '',
  provider: 'openai',
  modelType: 'embedding',
  modelName: '',
  baseUrl: '',
  apiKey: '',
  dimensions: 768,
  temperature: 0.1,
  maxTokens: 2048,
};

const GraphModels: React.FC = () => {
  const [filter, setFilter] = useState<string>('');
  const [showModal, setShowModal] = useState(false);
  const [editingModel, setEditingModel] = useState<ModelConfig | null>(null);
  const [form, setForm] = useState<CreateModelRequest>({ ...emptyForm });
  const [testResults, setTestResults] = useState<Record<string, { status: string; message: string }>>({});
  const [availableModels, setAvailableModels] = useState<{ id: string; ownedBy?: string }[]>([]);

  const { data, isLoading } = useListModelsQuery(filter ? { type: filter } : undefined);
  const [createModel, { isLoading: creating }] = useCreateModelMutation();
  const [updateModel, { isLoading: updating }] = useUpdateModelMutation();
  const [deleteModel] = useDeleteModelMutation();
  const [setDefaultModel] = useSetDefaultModelMutation();
  const [testModel] = useTestModelMutation();
  const [fetchProviderModels, { isLoading: fetching }] = useFetchProviderModelsMutation();

  const models = data?.models ?? [];
  const saving = creating || updating;

  const openCreate = () => {
    setEditingModel(null);
    setForm({ ...emptyForm });
    setAvailableModels([]);
    setShowModal(true);
  };

  const openEdit = (model: ModelConfig) => {
    setEditingModel(model);
    setAvailableModels([]);
    setForm({
      name: model.name,
      provider: model.provider,
      modelType: model.modelType,
      modelName: model.modelName,
      baseUrl: model.baseUrl || '',
      apiKey: '',
      dimensions: model.dimensions || 768,
      temperature: model.temperature || 0.1,
      maxTokens: model.maxTokens || 2048,
    });
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
    setEditingModel(null);
  };

  const handleSave = useCallback(async () => {
    try {
      if (editingModel) {
        const body: UpdateModelRequest = {};
        if (form.name !== editingModel.name) body.name = form.name;
        if (form.baseUrl !== (editingModel.baseUrl || '')) body.baseUrl = form.baseUrl;
        if (form.apiKey) body.apiKey = form.apiKey;
        if (form.dimensions !== editingModel.dimensions) body.dimensions = form.dimensions;
        if (form.temperature !== editingModel.temperature) body.temperature = form.temperature;
        if (form.maxTokens !== editingModel.maxTokens) body.maxTokens = form.maxTokens;
        await updateModel({ uuid: editingModel.uuid, body }).unwrap();
      } else {
        await createModel(form).unwrap();
      }
      closeModal();
    } catch {
      // Handled by RTK Query
    }
  }, [createModel, updateModel, editingModel, form]);

  const handleFetchModels = useCallback(async () => {
    if (!form.baseUrl) return;
    try {
      const result = await fetchProviderModels({
        provider: form.provider,
        baseUrl: form.baseUrl,
        apiKey: form.apiKey || undefined,
      }).unwrap();
      setAvailableModels(result.models ?? []);
      // Auto-select first if modelName is empty
      if (!form.modelName && result.models?.length) {
        setForm(prev => ({ ...prev, modelName: result.models[0].id }));
      }
    } catch {
      setAvailableModels([]);
    }
  }, [fetchProviderModels, form.provider, form.baseUrl, form.apiKey, form.modelName]);

  const handleDelete = useCallback(async (uuid: string) => {
    if (!confirm('Delete this model configuration?')) return;
    try {
      await deleteModel(uuid).unwrap();
    } catch {
      // Handled by RTK Query
    }
  }, [deleteModel]);

  const handleTest = useCallback(async (uuid: string) => {
    setTestResults(prev => ({ ...prev, [uuid]: { status: 'testing', message: 'Testing...' } }));
    try {
      const result = await testModel(uuid).unwrap();
      setTestResults(prev => ({ ...prev, [uuid]: result }));
    } catch {
      setTestResults(prev => ({ ...prev, [uuid]: { status: 'error', message: 'Test failed' } }));
    }
  }, [testModel]);

  const updateForm = (field: string, value: unknown) => {
    setForm(prev => ({ ...prev, [field]: value }));
  };

  const isEditing = !!editingModel;

  return (
    <>
      <Row className="g-3 mb-3">
        <Col>
          <div className="d-flex align-items-center justify-content-between">
            <h5 className="mb-0">AI Models</h5>
            <div className="d-flex gap-2">
              <Form.Select size="sm" value={filter} onChange={e => setFilter(e.target.value)} style={{ width: 150 }}>
                <option value="">All types</option>
                <option value="embedding">Embedding</option>
                <option value="llm">LLM</option>
              </Form.Select>
              <Button size="sm" variant="primary" onClick={openCreate}>
                Add Model
              </Button>
            </div>
          </div>
        </Col>
      </Row>

      <Row className="g-3">
        <Col>
          <Card>
            <Card.Body className="p-0">
              {isLoading ? (
                <div className="text-center p-3"><Spinner size="sm" /></div>
              ) : models.length === 0 ? (
                <Alert variant="info" className="m-3 mb-0">No models configured. Add one to get started.</Alert>
              ) : (
                <Table size="sm" hover responsive className="mb-0">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Provider</th>
                      <th>Type</th>
                      <th>Model</th>
                      <th>Base URL</th>
                      <th>Details</th>
                      <th>Status</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {models.map(m => (
                      <tr key={m.uuid}>
                        <td className="fw-semibold">
                          {m.name}
                          {m.isDefault && <Badge bg="success" className="ms-2">Default</Badge>}
                        </td>
                        <td>
                          <Badge bg={m.provider === 'ollama' ? 'info' : 'warning'}>
                            {m.provider}
                          </Badge>
                        </td>
                        <td><Badge bg="secondary">{m.modelType}</Badge></td>
                        <td><code className="small">{m.modelName}</code></td>
                        <td className="small text-muted" style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {m.baseUrl || '-'}
                        </td>
                        <td className="small text-muted">
                          {m.modelType === 'embedding' && m.dimensions ? `${m.dimensions}d` : ''}
                          {m.modelType === 'llm' ? `temp: ${m.temperature ?? 0}, max: ${m.maxTokens ?? 0}` : ''}
                        </td>
                        <td>
                          {testResults[m.uuid] ? (
                            <Badge bg={testResults[m.uuid].status === 'ok' ? 'success' : testResults[m.uuid].status === 'testing' ? 'secondary' : 'danger'}>
                              {testResults[m.uuid].status === 'testing' ? <Spinner size="sm" /> : testResults[m.uuid].message}
                            </Badge>
                          ) : (
                            <Badge bg={m.isActive ? 'success' : 'secondary'} className="bg-opacity-25 text-body">
                              {m.isActive ? 'active' : 'inactive'}
                            </Badge>
                          )}
                        </td>
                        <td>
                          <div className="d-flex gap-1 flex-nowrap">
                            <Button variant="outline-primary" size="sm" onClick={() => handleTest(m.uuid)}>Test</Button>
                            <Button variant="outline-secondary" size="sm" onClick={() => openEdit(m)}>Edit</Button>
                            {!m.isDefault && (
                              <Button variant="outline-success" size="sm" onClick={() => setDefaultModel(m.uuid)}>
                                Default
                              </Button>
                            )}
                            <Button variant="outline-danger" size="sm" onClick={() => handleDelete(m.uuid)}>
                              Delete
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </Table>
              )}
            </Card.Body>
          </Card>
        </Col>
      </Row>

      {/* Create / Edit Model Modal */}
      <Modal show={showModal} onHide={closeModal} size="lg">
        <Modal.Header closeButton>
          <Modal.Title>{isEditing ? 'Edit Model' : 'Add AI Model'}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Row className="g-3">
            <Col md={6}>
              <Form.Group>
                <Form.Label className="small">Display Name</Form.Label>
                <Form.Control size="sm" value={form.name} onChange={e => updateForm('name', e.target.value)} placeholder="My Embedding Model" />
              </Form.Group>
            </Col>
            <Col md={3}>
              <Form.Group>
                <Form.Label className="small">Provider</Form.Label>
                <Form.Select size="sm" value={form.provider} onChange={e => updateForm('provider', e.target.value)} disabled={isEditing}>
                  <option value="openai">OpenAI / Compatible</option>
                  <option value="ollama">Ollama</option>
                </Form.Select>
              </Form.Group>
            </Col>
            <Col md={3}>
              <Form.Group>
                <Form.Label className="small">Type</Form.Label>
                <Form.Select size="sm" value={form.modelType} onChange={e => updateForm('modelType', e.target.value)} disabled={isEditing}>
                  <option value="embedding">Embedding</option>
                  <option value="llm">LLM</option>
                </Form.Select>
              </Form.Group>
            </Col>
            <Col md={6}>
              <Form.Group>
                <Form.Label className="small">Base URL</Form.Label>
                <div className="d-flex gap-2">
                  <Form.Control
                    size="sm"
                    value={form.baseUrl}
                    onChange={e => updateForm('baseUrl', e.target.value)}
                    placeholder={form.provider === 'ollama' ? 'http://host:11434' : 'http://host:8080/v1'}
                  />
                  <Button
                    size="sm"
                    variant="outline-secondary"
                    onClick={handleFetchModels}
                    disabled={fetching || !form.baseUrl}
                    style={{ whiteSpace: 'nowrap' }}
                  >
                    {fetching ? <Spinner size="sm" /> : 'Fetch Models'}
                  </Button>
                </div>
                <Form.Text className="text-muted">
                  {form.provider === 'openai'
                    ? 'For llama.cpp, vLLM, LocalAI, etc. Leave empty for OpenAI cloud.'
                    : 'Leave empty to use global Ollama config'}
                </Form.Text>
              </Form.Group>
            </Col>
            <Col md={6}>
              <Form.Group>
                <Form.Label className="small">Model Name</Form.Label>
                {availableModels.length > 0 ? (
                  <Form.Select
                    size="sm"
                    value={form.modelName}
                    onChange={e => updateForm('modelName', e.target.value)}
                    disabled={isEditing}
                  >
                    <option value="">Select a model...</option>
                    {availableModels.map(m => (
                      <option key={m.id} value={m.id}>
                        {m.id}{m.ownedBy ? ` (${m.ownedBy})` : ''}
                      </option>
                    ))}
                  </Form.Select>
                ) : (
                  <Form.Control
                    size="sm"
                    value={form.modelName}
                    onChange={e => updateForm('modelName', e.target.value)}
                    disabled={isEditing}
                    placeholder="Enter model name or click Fetch Models"
                  />
                )}
                <Form.Text className="text-muted">
                  Enter a base URL and click "Fetch Models" to see available models
                </Form.Text>
              </Form.Group>
            </Col>
            <Col md={6}>
              <Form.Group>
                <Form.Label className="small">API Key</Form.Label>
                <Form.Control
                  size="sm"
                  type="password"
                  value={form.apiKey}
                  onChange={e => updateForm('apiKey', e.target.value)}
                  placeholder={isEditing ? '(unchanged)' : 'sk-... or leave empty for local'}
                />
                <Form.Text className="text-muted">
                  Required for OpenAI cloud. Not needed for local servers (llama.cpp, Ollama).
                </Form.Text>
              </Form.Group>
            </Col>
            {form.modelType === 'embedding' && (
              <Col md={6}>
                <Form.Group>
                  <Form.Label className="small">Dimensions</Form.Label>
                  <Form.Control size="sm" type="number" value={form.dimensions} onChange={e => updateForm('dimensions', parseInt(e.target.value) || 0)} />
                  <Form.Text className="text-muted">Vector size output by the model</Form.Text>
                </Form.Group>
              </Col>
            )}
            {form.modelType === 'llm' && (
              <>
                <Col md={3}>
                  <Form.Group>
                    <Form.Label className="small">Temperature</Form.Label>
                    <Form.Control size="sm" type="number" step="0.1" min="0" max="2" value={form.temperature} onChange={e => updateForm('temperature', parseFloat(e.target.value) || 0)} />
                  </Form.Group>
                </Col>
                <Col md={3}>
                  <Form.Group>
                    <Form.Label className="small">Max Tokens</Form.Label>
                    <Form.Control size="sm" type="number" value={form.maxTokens} onChange={e => updateForm('maxTokens', parseInt(e.target.value) || 0)} />
                  </Form.Group>
                </Col>
              </>
            )}
          </Row>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" size="sm" onClick={closeModal}>Cancel</Button>
          <Button variant="primary" size="sm" onClick={handleSave} disabled={saving || !form.name || (!isEditing && !form.modelName)}>
            {saving ? <Spinner size="sm" /> : isEditing ? 'Save Changes' : 'Create'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default GraphModels;
