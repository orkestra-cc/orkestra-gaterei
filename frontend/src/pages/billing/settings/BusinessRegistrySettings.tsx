import React, { useState, useEffect } from 'react';
import { Card, Form, Button, Alert, Row, Col, Spinner } from 'react-bootstrap';
import {
  useGetBusinessRegistryConfigQuery,
  useLazyGetBusinessRegistryConfigQuery,
  useConfigureBusinessRegistryMutation,
  type ConfigureBusinessRegistryInput,
} from 'store/api/billingApi';
import PageHeader from 'components/common/PageHeader';
import FalconCardHeader from 'components/common/FalconCardHeader';

const BusinessRegistrySettings: React.FC = () => {
  // Default fiscal ID from environment or empty
  const defaultFiscalId = import.meta.env.VITE_OPENAPI_FISCAL_ID || '';

  const [fiscalId, setFiscalId] = useState(defaultFiscalId);
  const [email, setEmail] = useState('');
  const [applySignature, setApplySignature] = useState(false);
  const [applyLegalStorage, setApplyLegalStorage] = useState(false);

  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');

  // Query for existing configuration
  const [getConfig, { data: existingConfig, isLoading: isLoadingConfig, isError: isConfigError }] =
    useLazyGetBusinessRegistryConfigQuery();

  // Mutation for saving configuration
  const [configureRegistry, { isLoading: isSaving }] = useConfigureBusinessRegistryMutation();

  // Load existing config when fiscal ID changes
  useEffect(() => {
    if (fiscalId && fiscalId.length >= 11) {
      getConfig(fiscalId);
    }
  }, [fiscalId, getConfig]);

  // Populate form with existing config
  useEffect(() => {
    if (existingConfig) {
      setEmail(existingConfig.email || '');
      setApplySignature(existingConfig.applySignature || false);
      setApplyLegalStorage(existingConfig.applyLegalStorage || false);
    }
  }, [existingConfig]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    // Validation
    if (!fiscalId || fiscalId.length < 11) {
      setError('Fiscal ID (P.IVA) deve essere di almeno 11 caratteri');
      return;
    }
    if (!email || !email.includes('@')) {
      setError('Email non valida');
      return;
    }

    const input: ConfigureBusinessRegistryInput = {
      fiscalId,
      email,
      applySignature,
      applyLegalStorage,
    };

    try {
      const result = await configureRegistry(input).unwrap();
      if (result.success) {
        setSuccess(result.message || 'Configurazione salvata con successo');
        // Refresh the config
        getConfig(fiscalId);
      } else {
        setError(result.message || 'Errore durante il salvataggio');
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message :
        (err as { data?: { detail?: string } })?.data?.detail || 'Errore durante il salvataggio della configurazione';
      setError(errorMessage);
    }
  };

  const isConfigured = existingConfig?.active;

  return (
    <>
      <PageHeader
        title="Configurazione Business Registry"
        description="Configura l'anagrafica aziendale per l'invio delle fatture elettroniche tramite OpenAPI SDI"
        className="mb-3"
      />

      <Card className="mb-3">
        <FalconCardHeader
          title="Anagrafica OpenAPI SDI"
          titleTag="h5"
          endEl={
            isConfigured ? (
              <span className="badge bg-success">Configurato</span>
            ) : (
              <span className="badge bg-warning text-dark">Non configurato</span>
            )
          }
        />
        <Card.Body>
          {error && (
            <Alert variant="danger" dismissible onClose={() => setError('')}>
              {error}
            </Alert>
          )}
          {success && (
            <Alert variant="success" dismissible onClose={() => setSuccess('')}>
              {success}
            </Alert>
          )}

          <Form onSubmit={handleSubmit}>
            <Row className="mb-3">
              <Col md={6}>
                <Form.Group>
                  <Form.Label>
                    Fiscal ID (P.IVA) <span className="text-danger">*</span>
                  </Form.Label>
                  <Form.Control
                    type="text"
                    value={fiscalId}
                    onChange={(e) => setFiscalId(e.target.value)}
                    placeholder="es. 02081880490"
                    maxLength={16}
                    required
                  />
                  <Form.Text className="text-muted">
                    La partita IVA dell'azienda emittente
                  </Form.Text>
                </Form.Group>
              </Col>
              <Col md={6}>
                <Form.Group>
                  <Form.Label>
                    Email per notifiche SDI <span className="text-danger">*</span>
                  </Form.Label>
                  <Form.Control
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="es. fatture@azienda.it"
                    required
                  />
                  <Form.Text className="text-muted">
                    Email per ricevere le notifiche dallo SDI
                  </Form.Text>
                </Form.Group>
              </Col>
            </Row>

            <Row className="mb-3">
              <Col md={6}>
                <Form.Check
                  type="switch"
                  id="applySignature"
                  label="Applica firma digitale"
                  checked={applySignature}
                  onChange={(e) => setApplySignature(e.target.checked)}
                />
                <Form.Text className="text-muted d-block mt-1">
                  Firma digitalmente le fatture prima dell'invio
                </Form.Text>
              </Col>
              <Col md={6}>
                <Form.Check
                  type="switch"
                  id="applyLegalStorage"
                  label="Conservazione sostitutiva"
                  checked={applyLegalStorage}
                  onChange={(e) => setApplyLegalStorage(e.target.checked)}
                />
                <Form.Text className="text-muted d-block mt-1">
                  Archivia le fatture in conservazione a norma
                </Form.Text>
              </Col>
            </Row>

            {existingConfig && (
              <Alert variant="info" className="mb-3">
                <strong>Configurazione esistente:</strong>
                <ul className="mb-0 mt-2">
                  <li>Email: {existingConfig.email}</li>
                  <li>Firma digitale: {existingConfig.applySignature ? 'Attiva' : 'Non attiva'}</li>
                  <li>Conservazione: {existingConfig.applyLegalStorage ? 'Attiva' : 'Non attiva'}</li>
                  <li>Stato: {existingConfig.active ? 'Attivo' : 'Non attivo'}</li>
                </ul>
              </Alert>
            )}

            <div className="d-flex justify-content-end gap-2">
              <Button
                variant="primary"
                type="submit"
                disabled={isSaving || isLoadingConfig}
              >
                {isSaving ? (
                  <>
                    <Spinner size="sm" className="me-2" />
                    Salvataggio...
                  </>
                ) : (
                  'Salva configurazione'
                )}
              </Button>
            </div>
          </Form>
        </Card.Body>
      </Card>

      <Card>
        <FalconCardHeader title="Informazioni" titleTag="h5" />
        <Card.Body>
          <p className="mb-2">
            <strong>Importante:</strong> La configurazione del Business Registry è necessaria prima di poter
            inviare fatture elettroniche tramite OpenAPI SDI.
          </p>
          <p className="mb-2">
            Senza questa configurazione, l'invio delle fatture fallirà con l'errore:{' '}
            <code>Missing configuration for fiscal Id</code>
          </p>
          <p className="mb-0">
            Per maggiori informazioni, consulta la{' '}
            <a
              href="https://console.openapi.com/apis/sdi/documentation"
              target="_blank"
              rel="noopener noreferrer"
            >
              documentazione OpenAPI SDI
            </a>
            .
          </p>
        </Card.Body>
      </Card>
    </>
  );
};

export default BusinessRegistrySettings;
