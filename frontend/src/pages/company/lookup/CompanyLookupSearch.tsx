import { useState } from 'react';
import { Card, Form, Button, Row, Col, Alert, Spinner } from 'react-bootstrap';
import { useLazyLookupCompanyQuery } from 'store/api/companyApi';
import SubtleBadge from 'components/common/SubtleBadge';
import type { BadgeColor } from 'components/common/SubtleBadge';
import { ACTIVITY_STATUS_COLORS, ACTIVITY_STATUS_LABELS } from 'types/company';
import { formatItalianDate } from 'types/billing';

const CompanyLookupSearch = () => {
  const [taxCode, setTaxCode] = useState('');
  const [trigger, { data: result, isFetching, error, isSuccess }] = useLazyLookupCompanyQuery();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = taxCode.trim();
    if (trimmed.length >= 11 && trimmed.length <= 16) {
      trigger(trimmed);
    }
  };

  const isValidLength = taxCode.trim().length >= 11 && taxCode.trim().length <= 16;

  const getErrorMessage = () => {
    if (!error) return null;
    if ('status' in error && error.status === 404) {
      return 'Azienda non trovata. Verifica il codice fiscale o la partita IVA inserita.';
    }
    return 'Errore durante la ricerca. Riprova più tardi.';
  };

  return (
    <Card>
      <Card.Header>
        <h6 className="mb-0">Cerca Azienda per Codice Fiscale o Partita IVA</h6>
      </Card.Header>
      <Card.Body>
        <Form onSubmit={handleSubmit}>
          <Row className="align-items-end g-3">
            <Col sm={8} md={6} lg={4}>
              <Form.Group>
                <Form.Label className="fs-9">Codice Fiscale / P.IVA</Form.Label>
                <Form.Control
                  type="text"
                  placeholder="Es. 12485671007"
                  value={taxCode}
                  onChange={(e) => setTaxCode(e.target.value.toUpperCase())}
                  className="font-monospace"
                  maxLength={16}
                />
              </Form.Group>
            </Col>
            <Col xs="auto">
              <Button
                type="submit"
                variant="primary"
                disabled={!isValidLength || isFetching}
              >
                {isFetching ? (
                  <>
                    <Spinner size="sm" className="me-2" />
                    Ricerca...
                  </>
                ) : (
                  'Cerca'
                )}
              </Button>
            </Col>
          </Row>
        </Form>

        {error && (
          <Alert variant="warning" className="mt-3 mb-0">
            {getErrorMessage()}
          </Alert>
        )}

        {isSuccess && result && (
          <Card className="mt-3 border">
            <Card.Body>
              <Row>
                <Col md={8}>
                  <div className="d-flex align-items-center mb-3">
                    <h5 className="mb-0 me-2">{result.companyName}</h5>
                    <SubtleBadge
                      bg={(ACTIVITY_STATUS_COLORS[result.activityStatus] || 'secondary') as BadgeColor}
                    >
                      {ACTIVITY_STATUS_LABELS[result.activityStatus] || result.activityStatus}
                    </SubtleBadge>
                  </div>
                </Col>
              </Row>
              <Row className="g-3">
                <Col sm={6} md={4}>
                  <div className="mb-2">
                    <small className="text-muted d-block">Codice Fiscale</small>
                    <span className="font-monospace fw-semibold">{result.taxCode}</span>
                  </div>
                </Col>
                <Col sm={6} md={4}>
                  <div className="mb-2">
                    <small className="text-muted d-block">Partita IVA</small>
                    <span className="font-monospace fw-semibold">{result.vatCode}</span>
                  </div>
                </Col>
                <Col sm={6} md={4}>
                  <div className="mb-2">
                    <small className="text-muted d-block">Codice SDI</small>
                    <span className="font-monospace fw-semibold">
                      {result.sdiCode || '-'}
                    </span>
                  </div>
                </Col>
                <Col sm={6} md={4}>
                  <div className="mb-2">
                    <small className="text-muted d-block">Indirizzo</small>
                    <span>
                      {result.address.street}
                      {result.address.streetNumber ? ` ${result.address.streetNumber}` : ''}
                    </span>
                  </div>
                </Col>
                <Col sm={6} md={4}>
                  <div className="mb-2">
                    <small className="text-muted d-block">Sede</small>
                    <span>
                      {result.address.zipCode} {result.address.town}
                      {result.address.province ? ` (${result.address.province})` : ''}
                    </span>
                  </div>
                </Col>
                <Col sm={6} md={4}>
                  <div className="mb-2">
                    <small className="text-muted d-block">Data Registrazione</small>
                    <span>
                      {result.registrationDate
                        ? formatItalianDate(result.registrationDate)
                        : '-'}
                    </span>
                  </div>
                </Col>
              </Row>
            </Card.Body>
          </Card>
        )}
      </Card.Body>
    </Card>
  );
};

export default CompanyLookupSearch;
