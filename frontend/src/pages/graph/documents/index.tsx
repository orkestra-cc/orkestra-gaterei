import { useState, useCallback, useRef } from 'react';
import { Row, Col, Card, Button, Form, Table, Badge, Spinner, Alert, Modal } from 'react-bootstrap';
import {
  useListDocumentsQuery,
  useUploadDocumentMutation,
  useDeleteDocumentMutation,
} from '../../../store/api/ragApi';

const statusColors: Record<string, string> = {
  pending: 'warning',
  processing: 'info',
  completed: 'success',
  failed: 'danger',
};

const GraphDocuments: React.FC = () => {
  const [statusFilter, setStatusFilter] = useState('');
  const [isoFilter, setIsoFilter] = useState('');
  const [showUpload, setShowUpload] = useState(false);
  const [uploadTitle, setUploadTitle] = useState('');
  const [uploadISO, setUploadISO] = useState('');
  const [uploadVersion, setUploadVersion] = useState('');
  const [uploadChunkSize, setUploadChunkSize] = useState(512);
  const [uploadChunkOverlap, setUploadChunkOverlap] = useState(50);
  const fileRef = useRef<HTMLInputElement>(null);

  const { data, isLoading, refetch } = useListDocumentsQuery(
    { status: statusFilter || undefined, isoStandard: isoFilter || undefined } as { status?: string; isoStandard?: string }
  );
  const [uploadDocument, { isLoading: uploading }] = useUploadDocumentMutation();
  const [deleteDocument] = useDeleteDocumentMutation();

  const documents = data?.documents ?? [];

  // Auto-refresh while any document is processing
  const hasProcessing = documents.some(d => d.status === 'pending' || d.status === 'processing');
  if (hasProcessing) {
    setTimeout(() => refetch(), 3000);
  }

  const handleUpload = useCallback(async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('file', file);
    formData.append('title', uploadTitle || file.name);
    if (uploadISO) formData.append('isoStandard', uploadISO);
    if (uploadVersion) formData.append('version', uploadVersion);
    formData.append('chunkSize', String(uploadChunkSize));
    formData.append('chunkOverlap', String(uploadChunkOverlap));

    try {
      await uploadDocument(formData).unwrap();
      setShowUpload(false);
      setUploadTitle('');
      setUploadISO('');
      setUploadVersion('');
      if (fileRef.current) fileRef.current.value = '';
    } catch {
      // Handled by RTK Query
    }
  }, [uploadDocument, uploadTitle, uploadISO, uploadVersion, uploadChunkSize, uploadChunkOverlap]);

  const handleDelete = useCallback(async (uuid: string) => {
    if (!confirm('Delete this document and all its chunks from the knowledge graph?')) return;
    try {
      await deleteDocument(uuid).unwrap();
    } catch {
      // Handled by RTK Query
    }
  }, [deleteDocument]);

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <>
      <Row className="g-3 mb-3">
        <Col>
          <div className="d-flex align-items-center justify-content-between">
            <h5 className="mb-0">Documents</h5>
            <div className="d-flex gap-2">
              <Form.Select size="sm" value={statusFilter} onChange={e => setStatusFilter(e.target.value)} style={{ width: 130 }}>
                <option value="">All status</option>
                <option value="pending">Pending</option>
                <option value="processing">Processing</option>
                <option value="completed">Completed</option>
                <option value="failed">Failed</option>
              </Form.Select>
              <Form.Control
                size="sm"
                placeholder="ISO filter..."
                value={isoFilter}
                onChange={e => setIsoFilter(e.target.value)}
                style={{ width: 130 }}
              />
              <Button size="sm" variant="primary" onClick={() => setShowUpload(true)}>
                Upload Document
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
              ) : documents.length === 0 ? (
                <Alert variant="info" className="m-3 mb-0">No documents ingested yet. Upload one to get started.</Alert>
              ) : (
                <Table size="sm" hover responsive className="mb-0">
                  <thead>
                    <tr>
                      <th>Title</th>
                      <th>File</th>
                      <th>ISO</th>
                      <th>Status</th>
                      <th>Chunks</th>
                      <th>Size</th>
                      <th>Date</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {documents.map(d => (
                      <tr key={d.uuid}>
                        <td className="fw-semibold">{d.title}</td>
                        <td className="small text-muted">{d.fileName}</td>
                        <td>{d.isoStandard ? <Badge bg="primary">{d.isoStandard}</Badge> : '-'}</td>
                        <td>
                          <Badge bg={statusColors[d.status] || 'secondary'}>
                            {d.status === 'processing' && <Spinner size="sm" className="me-1" />}
                            {d.status}
                          </Badge>
                          {d.error && (
                            <small className="d-block text-danger mt-1" style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                              {d.error}
                            </small>
                          )}
                        </td>
                        <td>{d.chunkCount || '-'}</td>
                        <td className="small text-muted">{formatSize(d.fileSize)}</td>
                        <td className="small text-muted">{new Date(d.createdAt).toLocaleDateString()}</td>
                        <td>
                          <Button variant="outline-danger" size="sm" onClick={() => handleDelete(d.uuid)}>
                            Delete
                          </Button>
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

      {/* Upload Modal */}
      <Modal show={showUpload} onHide={() => setShowUpload(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Upload Document</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group className="mb-3">
            <Form.Label className="small">File (PDF or Text)</Form.Label>
            <Form.Control type="file" size="sm" ref={fileRef} accept=".pdf,.txt,.md,.text" />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label className="small">Title</Form.Label>
            <Form.Control size="sm" value={uploadTitle} onChange={e => setUploadTitle(e.target.value)} placeholder="Leave empty to use filename" />
          </Form.Group>
          <Row className="g-2 mb-3">
            <Col>
              <Form.Group>
                <Form.Label className="small">ISO Standard</Form.Label>
                <Form.Control size="sm" value={uploadISO} onChange={e => setUploadISO(e.target.value)} placeholder="e.g. ISO 9001" />
              </Form.Group>
            </Col>
            <Col>
              <Form.Group>
                <Form.Label className="small">Version</Form.Label>
                <Form.Control size="sm" value={uploadVersion} onChange={e => setUploadVersion(e.target.value)} placeholder="e.g. 2015" />
              </Form.Group>
            </Col>
          </Row>
          <Row className="g-2">
            <Col>
              <Form.Group>
                <Form.Label className="small">Chunk Size</Form.Label>
                <Form.Control size="sm" type="number" value={uploadChunkSize} onChange={e => setUploadChunkSize(parseInt(e.target.value) || 512)} />
              </Form.Group>
            </Col>
            <Col>
              <Form.Group>
                <Form.Label className="small">Chunk Overlap</Form.Label>
                <Form.Control size="sm" type="number" value={uploadChunkOverlap} onChange={e => setUploadChunkOverlap(parseInt(e.target.value) || 50)} />
              </Form.Group>
            </Col>
          </Row>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" size="sm" onClick={() => setShowUpload(false)}>Cancel</Button>
          <Button variant="primary" size="sm" onClick={handleUpload} disabled={uploading || !fileRef.current?.files?.length}>
            {uploading ? <><Spinner size="sm" className="me-1" /> Uploading...</> : 'Upload & Ingest'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default GraphDocuments;
