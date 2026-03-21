import { useState, useCallback, useRef, useEffect } from 'react';
import { Row, Col, Card, Button, Form, Spinner, Badge, Accordion } from 'react-bootstrap';
import { useRagQueryMutation } from '../../../store/api/ragApi';
import type { RagQueryResponse, SourceRef } from '../../../types/rag';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  sources?: SourceRef[];
  metadata?: RagQueryResponse['metadata'];
}

const GraphRAG: React.FC = () => {
  const [question, setQuestion] = useState('');
  const [isoFilter, setIsoFilter] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const [ragQuery, { isLoading }] = useRagQueryMutation();

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSubmit = useCallback(async () => {
    if (!question.trim() || isLoading) return;

    const q = question.trim();
    setQuestion('');

    // Add user message
    setMessages(prev => [...prev, { role: 'user', content: q }]);

    try {
      const result = await ragQuery({
        question: q,
        isoStandard: isoFilter || undefined,
      }).unwrap();

      setMessages(prev => [...prev, {
        role: 'assistant',
        content: result.answer,
        sources: result.sources,
        metadata: result.metadata,
      }]);
    } catch {
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: 'An error occurred while processing your question. Please try again.',
      }]);
    }
  }, [question, isoFilter, isLoading, ragQuery]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <>
      <Row className="g-3 mb-3">
        <Col>
          <div className="d-flex align-items-center justify-content-between">
            <h5 className="mb-0">RAG Query</h5>
            <div className="d-flex gap-2 align-items-center">
              <Form.Label className="mb-0 small text-muted">ISO Filter:</Form.Label>
              <Form.Control
                size="sm"
                value={isoFilter}
                onChange={e => setIsoFilter(e.target.value)}
                placeholder="All standards"
                style={{ width: 140 }}
              />
              <Button size="sm" variant="outline-secondary" onClick={() => setMessages([])}>
                Clear
              </Button>
            </div>
          </div>
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        <Col>
          <Card style={{ height: 'calc(100vh - 280px)', display: 'flex', flexDirection: 'column' }}>
            {/* Messages */}
            <Card.Body className="flex-grow-1 overflow-auto p-3">
              {messages.length === 0 ? (
                <div className="text-center text-muted py-5">
                  <p className="mb-1">Ask questions about your ISO norms and documents.</p>
                  <small>Example: "What are the requirements for risk assessment in ISO 27001?"</small>
                </div>
              ) : (
                messages.map((msg, i) => (
                  <div key={i} className={`mb-3 d-flex ${msg.role === 'user' ? 'justify-content-end' : 'justify-content-start'}`}>
                    <div
                      className={`p-3 rounded-3 ${msg.role === 'user' ? 'bg-primary text-white' : 'bg-light border'}`}
                      style={{ maxWidth: '80%' }}
                    >
                      <div style={{ whiteSpace: 'pre-wrap' }}>{msg.content}</div>

                      {/* Sources */}
                      {msg.sources && msg.sources.length > 0 && (
                        <Accordion className="mt-2" flush>
                          <Accordion.Item eventKey="0">
                            <Accordion.Header>
                              <small>{msg.sources.length} source{msg.sources.length > 1 ? 's' : ''}</small>
                            </Accordion.Header>
                            <Accordion.Body className="p-2">
                              {msg.sources.map((src, j) => (
                                <div key={j} className="border-bottom pb-2 mb-2 small">
                                  <div className="d-flex gap-1 mb-1 flex-wrap">
                                    <Badge bg="primary">{src.documentTitle}</Badge>
                                    {src.isoStandard && <Badge bg="info">{src.isoStandard}</Badge>}
                                    {src.sectionTitle && <Badge bg="secondary">{src.sectionTitle}</Badge>}
                                    <Badge bg="success">{(src.score * 100).toFixed(0)}%</Badge>
                                  </div>
                                  <div className="text-muted" style={{ maxHeight: 80, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                    {src.chunkText}
                                  </div>
                                </div>
                              ))}
                            </Accordion.Body>
                          </Accordion.Item>
                        </Accordion>
                      )}

                      {/* Timing metadata */}
                      {msg.metadata && (
                        <div className="mt-2 d-flex gap-2 flex-wrap">
                          <small className="text-muted">embed: {msg.metadata.embeddingTimeMs}ms</small>
                          <small className="text-muted">search: {msg.metadata.searchTimeMs}ms</small>
                          <small className="text-muted">llm: {msg.metadata.llmTimeMs}ms</small>
                          <small className="text-muted">total: {msg.metadata.totalTimeMs}ms</small>
                          <small className="text-muted">chunks: {msg.metadata.chunksRetrieved}</small>
                        </div>
                      )}
                    </div>
                  </div>
                ))
              )}
              {isLoading && (
                <div className="d-flex justify-content-start mb-3">
                  <div className="bg-light border p-3 rounded-3">
                    <Spinner size="sm" className="me-2" />
                    Thinking...
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </Card.Body>

            {/* Input */}
            <Card.Footer className="p-2">
              <div className="d-flex gap-2">
                <Form.Control
                  as="textarea"
                  rows={1}
                  size="sm"
                  value={question}
                  onChange={e => setQuestion(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="Ask a question about your ISO documents..."
                  disabled={isLoading}
                  style={{ resize: 'none' }}
                />
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSubmit}
                  disabled={isLoading || !question.trim()}
                  style={{ whiteSpace: 'nowrap' }}
                >
                  {isLoading ? <Spinner size="sm" /> : 'Send'}
                </Button>
              </div>
            </Card.Footer>
          </Card>
        </Col>
      </Row>
    </>
  );
};

export default GraphRAG;
