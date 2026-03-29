import { useState, useEffect } from 'react';
import { Row, Col, Card, Table, Badge, Button, Spinner, Form, Alert } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faScroll, faSave, faUndo, faArrowLeft, faPen, faRobot, faMagic } from '@fortawesome/free-solid-svg-icons';
import Background from 'components/common/Background';
import greetingsBg from 'assets/img/illustrations/ticket-greetings-bg.png';
import {
  useListSalesPromptsQuery,
  useGetSalesPromptQuery,
  useUpdateSalesPromptMutation,
  useResetSalesPromptMutation,
} from '../../../store/api/salesApi';
import type { SalesPromptConfig } from '../../../store/api/salesApi';

const CATEGORY_ICONS: Record<string, any> = {
  agents: faRobot,
  skills: faMagic,
};

const CATEGORY_COLORS: Record<string, string> = {
  agents: 'primary',
  skills: 'info',
};

// ─── Prompt Editor ───

function PromptEditor({ promptId, onBack }: { promptId: string; onBack: () => void }) {
  const { data: prompt, isLoading } = useGetSalesPromptQuery(promptId);
  const [updatePrompt, { isLoading: saving }] = useUpdateSalesPromptMutation();
  const [resetPrompt, { isLoading: resetting }] = useResetSalesPromptMutation();

  const [content, setContent] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [saved, setSaved] = useState(false);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (prompt) {
      setContent(prompt.content);
      setDisplayName(prompt.displayName);
      setDescription(prompt.description);
      setDirty(false);
    }
  }, [prompt]);

  if (isLoading || !prompt) {
    return <Card><Card.Body className="text-center py-5"><Spinner /></Card.Body></Card>;
  }

  const handleSave = async () => {
    setSaved(false);
    await updatePrompt({ uuid: prompt.uuid, content, displayName, description }).unwrap();
    setSaved(true);
    setDirty(false);
    setTimeout(() => setSaved(false), 3000);
  };

  const handleReset = async () => {
    if (!window.confirm('Reset this prompt to the built-in default? Your custom edits will be lost.')) return;
    const result = await resetPrompt(prompt.uuid).unwrap();
    setContent(result.content);
    setDirty(false);
  };

  // Count template variables
  const templateVars = (content.match(/\{\{\.(\w+)\}\}/g) || [])
    .map(v => v.replace(/\{\{\.|}\}/g, ''))
    .filter((v, i, a) => a.indexOf(v) === i);

  return (
    <>
      <Row className="g-3 mb-3">
        <Col xxl={12}>
          <Card>
            <Card.Header className="d-flex justify-content-between align-items-center">
              <div className="d-flex align-items-center gap-2">
                <Button variant="outline-secondary" size="sm" onClick={onBack}>
                  <FontAwesomeIcon icon={faArrowLeft} />
                </Button>
                <div>
                  <h5 className="mb-0 d-flex align-items-center gap-2">
                    <FontAwesomeIcon icon={CATEGORY_ICONS[prompt.category] || faScroll} className="text-primary" />
                    {prompt.displayName}
                  </h5>
                  <small className="text-muted">{prompt.category}/{prompt.name}</small>
                </div>
              </div>
              <div className="d-flex align-items-center gap-2">
                {prompt.isCustom && <Badge bg="warning">Customized</Badge>}
                {dirty && <Badge bg="secondary">Unsaved changes</Badge>}
                {saved && <Alert variant="success" className="mb-0 py-1 px-3 d-inline">Saved!</Alert>}
                <Button variant="outline-secondary" size="sm" onClick={handleReset} disabled={resetting || !prompt.isCustom}>
                  {resetting ? <Spinner size="sm" /> : <FontAwesomeIcon icon={faUndo} className="me-1" />}
                  Reset Default
                </Button>
                <Button variant="primary" size="sm" onClick={handleSave} disabled={saving || !dirty}>
                  {saving ? <Spinner size="sm" /> : <FontAwesomeIcon icon={faSave} className="me-1" />}
                  Save
                </Button>
              </div>
            </Card.Header>
          </Card>
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        <Col lg={8}>
          <Card>
            <Card.Header className="d-flex justify-content-between align-items-center">
              <h6 className="mb-0">Prompt Template</h6>
              <small className="text-muted">{content.length} chars | {content.split('\n').length} lines</small>
            </Card.Header>
            <Card.Body className="p-0">
              <Form.Control
                as="textarea"
                value={content}
                onChange={e => { setContent(e.target.value); setDirty(true); }}
                style={{
                  fontFamily: 'monospace',
                  fontSize: '0.85rem',
                  minHeight: '60vh',
                  border: 'none',
                  borderRadius: 0,
                  resize: 'vertical',
                }}
                className="p-3"
              />
            </Card.Body>
          </Card>
        </Col>

        <Col lg={4}>
          <Card className="mb-3">
            <Card.Header><h6 className="mb-0">Prompt Info</h6></Card.Header>
            <Card.Body>
              <Form.Group className="mb-3">
                <Form.Label className="fw-semibold">Display Name</Form.Label>
                <Form.Control
                  value={displayName}
                  onChange={e => { setDisplayName(e.target.value); setDirty(true); }}
                />
              </Form.Group>
              <Form.Group className="mb-3">
                <Form.Label className="fw-semibold">Description</Form.Label>
                <Form.Control
                  as="textarea"
                  rows={2}
                  value={description}
                  onChange={e => { setDescription(e.target.value); setDirty(true); }}
                />
              </Form.Group>
              <dl className="mb-0 small">
                <dt>Category</dt>
                <dd><Badge bg={CATEGORY_COLORS[prompt.category] || 'secondary'}>{prompt.category}</Badge></dd>
                <dt>Internal Name</dt>
                <dd className="font-monospace">{prompt.name}</dd>
                <dt>Last Updated</dt>
                <dd>{new Date(prompt.updatedAt).toLocaleString()}</dd>
              </dl>
            </Card.Body>
          </Card>

          <Card>
            <Card.Header><h6 className="mb-0">Template Variables</h6></Card.Header>
            <Card.Body>
              {templateVars.length > 0 ? (
                <div className="d-flex flex-wrap gap-1">
                  {templateVars.map(v => (
                    <Badge key={v} bg="body-tertiary" text="dark" className="font-monospace">
                      {'{{.'}{v}{'}}'}
                    </Badge>
                  ))}
                </div>
              ) : (
                <small className="text-muted">No template variables detected</small>
              )}
              <hr />
              <small className="text-muted">
                Available variables: <code>URL</code>, <code>Locale</code>, <code>CompanyName</code>,
                <code>Industry</code>, <code>Description</code>, <code>RawText</code>,
                <code>TechStack</code>, <code>TeamMembers</code>, <code>SocialLinks</code>,
                <code>ContactInfo</code>, <code>AboutText</code>, <code>RegistryData</code>,
                <code>Context</code>
              </small>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </>
  );
}

// ─── Prompts List Page ───

const PromptsPage = () => {
  const [editingId, setEditingId] = useState<string | null>(null);
  const { data, isLoading } = useListSalesPromptsQuery({});

  const prompts = data?.prompts || [];
  const agents = prompts.filter(p => p.category === 'agents');
  const skills = prompts.filter(p => p.category === 'skills');

  if (editingId) {
    return <PromptEditor promptId={editingId} onBack={() => setEditingId(null)} />;
  }

  return (
    <>
      <Row className="g-3 mb-3">
        <Col xxl={12}>
          <Card className="bg-body-tertiary dark__bg-opacity-50 shadow-none h-100">
            <Background image={greetingsBg} className="bg-card d-none d-sm-block" />
            <Card.Header className="d-flex align-items-center z-1 p-0">
              <div className="bg-primary rounded-circle p-3 ms-3">
                <FontAwesomeIcon icon={faScroll} className="text-white" size="2x" />
              </div>
              <div className="ms-3">
                <h6 className="mb-1 text-primary">Sales Intelligence</h6>
                <h4 className="mb-0 text-primary fw-bold">
                  Prompt<span className="text-info fw-medium"> Templates</span>
                </h4>
              </div>
            </Card.Header>
          </Card>
        </Col>
      </Row>

      {isLoading ? (
        <div className="text-center py-5"><Spinner /></div>
      ) : (
        <>
          {/* Agent Prompts */}
          <Row className="g-3 mb-3">
            <Col lg={12}>
              <Card>
                <Card.Header>
                  <h5 className="mb-0">
                    <FontAwesomeIcon icon={faRobot} className="text-primary me-2" />
                    Agent Prompts
                    <small className="text-muted fw-normal ms-2">— Used during the 5-agent parallel prospect analysis</small>
                  </h5>
                </Card.Header>
                <Card.Body className="p-0">
                  <PromptTable prompts={agents} onEdit={setEditingId} />
                </Card.Body>
              </Card>
            </Col>
          </Row>

          {/* Skill Prompts */}
          <Row className="g-3 mb-3">
            <Col lg={12}>
              <Card>
                <Card.Header>
                  <h5 className="mb-0">
                    <FontAwesomeIcon icon={faMagic} className="text-info me-2" />
                    Skill Prompts
                    <small className="text-muted fw-normal ms-2">— Used by individual skill endpoints</small>
                  </h5>
                </Card.Header>
                <Card.Body className="p-0">
                  <PromptTable prompts={skills} onEdit={setEditingId} />
                </Card.Body>
              </Card>
            </Col>
          </Row>
        </>
      )}
    </>
  );
};

function PromptTable({ prompts, onEdit }: { prompts: SalesPromptConfig[]; onEdit: (id: string) => void }) {
  if (prompts.length === 0) {
    return <div className="text-center text-muted py-4">No prompts found</div>;
  }

  return (
    <Table hover responsive className="mb-0">
      <thead>
        <tr>
          <th>Name</th>
          <th>Description</th>
          <th>Status</th>
          <th>Size</th>
          <th>Updated</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {prompts.map(p => (
          <tr key={p.uuid} style={{ cursor: 'pointer' }} onClick={() => onEdit(p.uuid)}>
            <td>
              <strong>{p.displayName}</strong>
              <br />
              <small className="text-muted font-monospace">{p.name}</small>
            </td>
            <td><small className="text-muted">{p.description}</small></td>
            <td>
              {p.isCustom
                ? <Badge bg="warning">Customized</Badge>
                : <Badge bg="secondary">Default</Badge>
              }
            </td>
            <td><small>{p.content?.length || 0} chars</small></td>
            <td><small>{new Date(p.updatedAt).toLocaleDateString()}</small></td>
            <td>
              <Button variant="outline-primary" size="sm" onClick={e => { e.stopPropagation(); onEdit(p.uuid); }}>
                <FontAwesomeIcon icon={faPen} />
              </Button>
            </td>
          </tr>
        ))}
      </tbody>
    </Table>
  );
}

export default PromptsPage;
