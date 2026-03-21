import { useState, useCallback } from 'react';
import { Row, Col, Card, Button, Form, Table, Badge, Spinner, Alert, Accordion } from 'react-bootstrap';
import {
  useListProjectionsQuery,
  useCreateProjectionMutation,
  useDropProjectionMutation,
  useRunAlgorithmMutation,
  useListAlgorithmsQuery,
} from '../../../store/api/graphApi';
import ResultsTable from '../components/ResultsTable';
import type { QueryResult } from '../../../types/graph';

const GDSPlayground: React.FC = () => {
  const [database, setDatabase] = useState('');
  const [algorithmResult, setAlgorithmResult] = useState<QueryResult | null>(null);

  // Projections
  const { data: projData, isLoading: projLoading } = useListProjectionsQuery(database ? { database } : undefined);
  const [createProjection, { isLoading: creating }] = useCreateProjectionMutation();
  const [dropProjection] = useDropProjectionMutation();

  // Algorithms
  const { data: algoData } = useListAlgorithmsQuery();
  const [runAlgorithm, { isLoading: running }] = useRunAlgorithmMutation();

  // New projection form
  const [projName, setProjName] = useState('');
  const [projNodes, setProjNodes] = useState('*');
  const [projRels, setProjRels] = useState('*');

  // Algorithm form
  const [selectedAlgo, setSelectedAlgo] = useState('');
  const [algoMode, setAlgoMode] = useState<'stream' | 'stats' | 'mutate' | 'write'>('stream');
  const [algoProjection, setAlgoProjection] = useState('');
  const [algoConfig, setAlgoConfig] = useState('{}');

  const handleCreateProjection = useCallback(async () => {
    if (!projName) return;
    try {
      await createProjection({
        name: projName,
        nodeProjection: projNodes,
        relationshipProjection: projRels,
        database: database || undefined,
      }).unwrap();
      setProjName('');
    } catch {
      // Error handled by RTK Query
    }
  }, [createProjection, projName, projNodes, projRels, database]);

  const handleDropProjection = useCallback(async (name: string) => {
    try {
      await dropProjection({ name, database: database || undefined }).unwrap();
    } catch {
      // Error handled by RTK Query
    }
  }, [dropProjection, database]);

  const handleRunAlgorithm = useCallback(async () => {
    if (!selectedAlgo || !algoProjection) return;
    try {
      let config: Record<string, unknown> = {};
      try {
        config = JSON.parse(algoConfig);
      } catch {
        // Use empty config if JSON is invalid
      }
      const res = await runAlgorithm({
        algorithm: selectedAlgo,
        mode: algoMode,
        projectionName: algoProjection,
        config,
        database: database || undefined,
      }).unwrap();
      setAlgorithmResult(res);
    } catch {
      setAlgorithmResult(null);
    }
  }, [runAlgorithm, selectedAlgo, algoMode, algoProjection, algoConfig, database]);

  // Group algorithms by category
  const algosByCategory = algoData?.algorithms?.reduce((acc, algo) => {
    if (!acc[algo.category]) acc[algo.category] = [];
    acc[algo.category].push(algo);
    return acc;
  }, {} as Record<string, typeof algoData.algorithms>) ?? {};

  const projections = projData?.projections ?? [];

  return (
    <>
      <Row className="g-3 mb-3">
        <Col>
          <div className="d-flex align-items-center justify-content-between">
            <h5 className="mb-0">GDS Playground</h5>
            <Form.Group className="d-flex align-items-center gap-2">
              <Form.Label className="mb-0 small text-muted">Database:</Form.Label>
              <Form.Control
                size="sm"
                type="text"
                placeholder="default"
                value={database}
                onChange={e => setDatabase(e.target.value)}
                style={{ width: 150 }}
              />
            </Form.Group>
          </div>
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        {/* Graph Projections */}
        <Col lg={6}>
          <Card className="h-100">
            <Card.Header>
              <div className="d-flex align-items-center justify-content-between">
                <h6 className="mb-0">Graph Projections</h6>
                <Badge bg="secondary">{projections.length}</Badge>
              </div>
            </Card.Header>
            <Card.Body>
              {projLoading ? (
                <div className="text-center"><Spinner size="sm" /></div>
              ) : projections.length === 0 ? (
                <Alert variant="info" className="mb-3">No projections. Create one below to run algorithms.</Alert>
              ) : (
                <Table size="sm" hover className="mb-3">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Nodes</th>
                      <th>Rels</th>
                      <th>Memory</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {projections.map(p => (
                      <tr key={p.name}>
                        <td className="fw-semibold">{p.name}</td>
                        <td>{p.nodeCount}</td>
                        <td>{p.relationshipCount}</td>
                        <td><small>{p.memoryUsage}</small></td>
                        <td>
                          <Button
                            variant="outline-danger"
                            size="sm"
                            onClick={() => handleDropProjection(p.name)}
                          >
                            Drop
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </Table>
              )}

              {/* Create Projection Form */}
              <div className="border rounded p-2">
                <small className="text-muted fw-semibold d-block mb-2">Create Projection</small>
                <Form.Group className="mb-2">
                  <Form.Control size="sm" placeholder="Projection name" value={projName} onChange={e => setProjName(e.target.value)} />
                </Form.Group>
                <Row className="g-2 mb-2">
                  <Col>
                    <Form.Control size="sm" placeholder="Node projection (*)" value={projNodes} onChange={e => setProjNodes(e.target.value)} />
                  </Col>
                  <Col>
                    <Form.Control size="sm" placeholder="Rel projection (*)" value={projRels} onChange={e => setProjRels(e.target.value)} />
                  </Col>
                </Row>
                <Button size="sm" variant="primary" onClick={handleCreateProjection} disabled={creating || !projName}>
                  {creating ? <Spinner size="sm" /> : 'Create'}
                </Button>
              </div>
            </Card.Body>
          </Card>
        </Col>

        {/* Run Algorithm */}
        <Col lg={6}>
          <Card className="h-100">
            <Card.Header>
              <h6 className="mb-0">Run Algorithm</h6>
            </Card.Header>
            <Card.Body>
              <Form.Group className="mb-2">
                <Form.Label className="small">Algorithm</Form.Label>
                <Form.Select size="sm" value={selectedAlgo} onChange={e => setSelectedAlgo(e.target.value)}>
                  <option value="">Select algorithm...</option>
                  {Object.entries(algosByCategory).map(([category, algos]) => (
                    <optgroup key={category} label={category.charAt(0).toUpperCase() + category.slice(1)}>
                      {algos.map(a => (
                        <option key={a.name} value={a.name}>{a.name} - {a.description}</option>
                      ))}
                    </optgroup>
                  ))}
                </Form.Select>
              </Form.Group>

              <Row className="g-2 mb-2">
                <Col>
                  <Form.Group>
                    <Form.Label className="small">Projection</Form.Label>
                    <Form.Select size="sm" value={algoProjection} onChange={e => setAlgoProjection(e.target.value)}>
                      <option value="">Select projection...</option>
                      {projections.map(p => (
                        <option key={p.name} value={p.name}>{p.name}</option>
                      ))}
                    </Form.Select>
                  </Form.Group>
                </Col>
                <Col xs={4}>
                  <Form.Group>
                    <Form.Label className="small">Mode</Form.Label>
                    <Form.Select size="sm" value={algoMode} onChange={e => setAlgoMode(e.target.value as typeof algoMode)}>
                      <option value="stream">stream</option>
                      <option value="stats">stats</option>
                      <option value="mutate">mutate</option>
                      <option value="write">write</option>
                    </Form.Select>
                  </Form.Group>
                </Col>
              </Row>

              <Accordion className="mb-2">
                <Accordion.Item eventKey="0">
                  <Accordion.Header><small>Algorithm Configuration (JSON)</small></Accordion.Header>
                  <Accordion.Body className="p-2">
                    <Form.Control
                      as="textarea"
                      rows={3}
                      size="sm"
                      className="font-monospace"
                      value={algoConfig}
                      onChange={e => setAlgoConfig(e.target.value)}
                      placeholder='{"maxIterations": 20}'
                    />
                  </Accordion.Body>
                </Accordion.Item>
              </Accordion>

              <Button
                variant="primary"
                size="sm"
                onClick={handleRunAlgorithm}
                disabled={running || !selectedAlgo || !algoProjection}
              >
                {running ? <><Spinner size="sm" className="me-1" /> Running...</> : 'Run Algorithm'}
              </Button>
            </Card.Body>
          </Card>
        </Col>
      </Row>

      {/* Algorithm Results */}
      {algorithmResult && (
        <Row className="g-3">
          <Col>
            <ResultsTable result={algorithmResult} isLoading={running} />
          </Col>
        </Row>
      )}
    </>
  );
};

export default GDSPlayground;
