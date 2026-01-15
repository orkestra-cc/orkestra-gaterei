import React, { useState } from 'react';
import { useNavigate } from 'react-router';
import {
  Card,
  Form,
  Button,
  Alert,
  Tab,
  Nav,
  Row,
  Col,
  Table,
  InputGroup,
} from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  faPlus,
  faTrash,
  faSave,
  faPaperPlane,
  faArrowLeft,
} from '@fortawesome/free-solid-svg-icons';
import {
  useCreateInvoiceMutation,
  useSendInvoiceMutation,
  useGetCustomersQuery,
} from 'store/api/billingApi';
import type {
  CreateInvoiceInput,
  CreateInvoiceLineInput,
  CreatePaymentTermsInput,
  DocumentType,
  PaymentMethod,
  PaymentCondition,
  UnitOfMeasure,
  VATNature,
} from 'types/billing';
import PageHeader from 'components/common/PageHeader';
import FalconCardHeader from 'components/common/FalconCardHeader';

// Default empty line
const createEmptyLine = (): CreateInvoiceLineInput => ({
  description: '',
  quantity: 1,
  unitOfMeasure: 'PZ' as UnitOfMeasure,
  unitPrice: 0,
  vatRate: 22,
  vatNature: undefined,
  discounts: [],
  productCode: '',
  startDate: undefined,
  endDate: undefined,
});

// Document type options
const DOCUMENT_TYPES: { value: DocumentType; label: string }[] = [
  { value: 'TD01', label: 'TD01 - Fattura' },
  { value: 'TD02', label: 'TD02 - Acconto/Anticipo su fattura' },
  { value: 'TD04', label: 'TD04 - Nota di Credito' },
  { value: 'TD05', label: 'TD05 - Nota di Debito' },
  { value: 'TD06', label: 'TD06 - Parcella' },
  { value: 'TD24', label: 'TD24 - Fattura differita' },
  { value: 'TD25', label: 'TD25 - Fattura differita (lett. b)' },
];

// Payment method options
const PAYMENT_METHODS: { value: PaymentMethod; label: string }[] = [
  { value: 'MP01', label: 'MP01 - Contanti' },
  { value: 'MP02', label: 'MP02 - Assegno' },
  { value: 'MP05', label: 'MP05 - Bonifico' },
  { value: 'MP08', label: 'MP08 - Carta di pagamento' },
  { value: 'MP12', label: 'MP12 - RIBA' },
  { value: 'MP19', label: 'MP19 - SEPA Direct Debit' },
  { value: 'MP23', label: 'MP23 - PagoPA' },
];

// Payment condition options
const PAYMENT_CONDITIONS: { value: PaymentCondition; label: string }[] = [
  { value: 'TP01', label: 'TP01 - Pagamento a rate' },
  { value: 'TP02', label: 'TP02 - Pagamento completo' },
  { value: 'TP03', label: 'TP03 - Anticipo' },
];

// Unit of measure options
const UNITS_OF_MEASURE: { value: UnitOfMeasure; label: string }[] = [
  { value: 'PZ', label: 'PZ - Pezzo' },
  { value: 'KG', label: 'KG - Chilogrammo' },
  { value: 'LT', label: 'LT - Litro' },
  { value: 'MT', label: 'MT - Metro' },
  { value: 'MQ', label: 'MQ - Metro quadrato' },
  { value: 'H', label: 'H - Ora' },
  { value: 'GG', label: 'GG - Giorno' },
  { value: 'MESE', label: 'MESE - Mese' },
];

// VAT rates
const VAT_RATES = [0, 4, 5, 10, 22];

// VAT Nature options (for 0% VAT)
const VAT_NATURES: { value: VATNature; label: string }[] = [
  { value: 'N1', label: 'N1 - Escluse ex art.15' },
  { value: 'N2.1', label: 'N2.1 - Non soggette (artt. 7-7septies)' },
  { value: 'N2.2', label: 'N2.2 - Non soggette (altri casi)' },
  { value: 'N3.1', label: 'N3.1 - Non imponibili (esportazioni)' },
  { value: 'N3.5', label: 'N3.5 - Non imponibili (dichiarazioni intento)' },
  { value: 'N3.6', label: 'N3.6 - Non imponibili (altre)' },
  { value: 'N4', label: 'N4 - Esenti' },
  { value: 'N5', label: 'N5 - Regime del margine' },
  { value: 'N6.1', label: 'N6.1 - Reverse charge (rottami)' },
  { value: 'N6.9', label: 'N6.9 - Reverse charge (altri casi)' },
];

const NewIssuedInvoice: React.FC = () => {
  const navigate = useNavigate();
  const [createInvoice, { isLoading: isCreating }] = useCreateInvoiceMutation();
  const [sendInvoice, { isLoading: isSending }] = useSendInvoiceMutation();
  const { data: customersData } = useGetCustomersQuery({ pageSize: 100 });

  const [activeTab, setActiveTab] = useState('document');
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');

  // Form state
  const [documentType, setDocumentType] = useState<DocumentType>('TD01');
  const [number, setNumber] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const [customerId, setCustomerId] = useState('');
  const [lines, setLines] = useState<CreateInvoiceLineInput[]>([createEmptyLine()]);
  const [causale, setCausale] = useState<string[]>(['']);
  const [internalNotes, setInternalNotes] = useState('');
  const [legalStorageEnabled, setLegalStorageEnabled] = useState(true);
  const [signatureEnabled, setSignatureEnabled] = useState(true);

  // Payment terms
  const [paymentCondition, setPaymentCondition] = useState<PaymentCondition>('TP02');
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('MP05');
  const [paymentIban, setPaymentIban] = useState('');
  const [paymentDueDate, setPaymentDueDate] = useState('');

  const isLoading = isCreating || isSending;

  // Calculate totals
  const calculateLineTotals = (line: CreateInvoiceLineInput) => {
    const totalPrice = line.quantity * line.unitPrice;
    const vatAmount = totalPrice * (line.vatRate / 100);
    return { totalPrice, vatAmount };
  };

  const totals = lines.reduce(
    (acc, line) => {
      const { totalPrice, vatAmount } = calculateLineTotals(line);
      return {
        taxable: acc.taxable + totalPrice,
        vat: acc.vat + vatAmount,
        total: acc.total + totalPrice + vatAmount,
      };
    },
    { taxable: 0, vat: 0, total: 0 }
  );

  // Line handlers
  const handleAddLine = () => {
    setLines([...lines, createEmptyLine()]);
  };

  const handleRemoveLine = (index: number) => {
    if (lines.length > 1) {
      setLines(lines.filter((_, i) => i !== index));
    }
  };

  const handleLineChange = (
    index: number,
    field: keyof CreateInvoiceLineInput,
    value: string | number | undefined
  ) => {
    const newLines = [...lines];
    newLines[index] = { ...newLines[index], [field]: value };
    setLines(newLines);
  };

  // Causale handlers
  const handleAddCausale = () => {
    setCausale([...causale, '']);
  };

  const handleRemoveCausale = (index: number) => {
    if (causale.length > 1) {
      setCausale(causale.filter((_, i) => i !== index));
    }
  };

  const handleCausaleChange = (index: number, value: string) => {
    const newCausale = [...causale];
    newCausale[index] = value;
    setCausale(newCausale);
  };

  // Validation
  const validate = (): boolean => {
    if (!number.trim()) {
      setError('Il numero fattura è obbligatorio');
      setActiveTab('document');
      return false;
    }

    if (!customerId) {
      setError('Selezionare un cliente');
      setActiveTab('document');
      return false;
    }

    if (lines.length === 0) {
      setError('Aggiungere almeno una riga');
      setActiveTab('lines');
      return false;
    }

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (!line.description.trim()) {
        setError(`Riga ${i + 1}: la descrizione è obbligatoria`);
        setActiveTab('lines');
        return false;
      }
      if (line.quantity <= 0) {
        setError(`Riga ${i + 1}: la quantità deve essere maggiore di zero`);
        setActiveTab('lines');
        return false;
      }
      if (line.vatRate === 0 && !line.vatNature) {
        setError(`Riga ${i + 1}: selezionare la natura IVA per aliquota 0%`);
        setActiveTab('lines');
        return false;
      }
    }

    return true;
  };

  // Build invoice input
  const buildInvoiceInput = (): CreateInvoiceInput => {
    const paymentTerms: CreatePaymentTermsInput | undefined =
      paymentMethod
        ? {
            condition: paymentCondition,
            paymentMethod: paymentMethod,
            iban: paymentIban || undefined,
            dueDate: paymentDueDate || undefined,
          }
        : undefined;

    return {
      documentType,
      number,
      date,
      currency: 'EUR',
      customerId,
      lines: lines.map((line) => ({
        ...line,
        vatNature: line.vatRate === 0 ? line.vatNature : undefined,
      })),
      paymentTerms,
      causale: causale.filter((c) => c.trim()),
      internalNotes: internalNotes || undefined,
      legalStorageEnabled,
      signatureEnabled,
    };
  };

  // Save as draft
  const handleSaveDraft = async () => {
    setError('');
    setSuccess('');

    if (!validate()) return;

    try {
      const input = buildInvoiceInput();
      await createInvoice(input).unwrap();
      setSuccess('Fattura salvata come bozza');
      setTimeout(() => navigate('/billing/invoices/issued'), 1500);
    } catch (err: unknown) {
      const errorMessage = err && typeof err === 'object' && 'data' in err
        ? (err as { data?: { message?: string } }).data?.message
        : undefined;
      setError(errorMessage || 'Errore durante il salvataggio della fattura');
    }
  };

  // Save and send to SDI
  const handleSaveAndSend = async () => {
    setError('');
    setSuccess('');

    if (!validate()) return;

    try {
      const input = buildInvoiceInput();
      const invoice = await createInvoice(input).unwrap();

      // Now send to SDI
      await sendInvoice(invoice.id).unwrap();
      setSuccess('Fattura creata e inviata al SDI');
      setTimeout(() => navigate('/billing/invoices/issued'), 1500);
    } catch (err: unknown) {
      const errorMessage = err && typeof err === 'object' && 'data' in err
        ? (err as { data?: { message?: string } }).data?.message
        : undefined;
      setError(errorMessage || 'Errore durante la creazione/invio della fattura');
    }
  };

  const selectedCustomer = customersData?.customers?.find((c) => c.id === customerId);

  return (
    <>
      <PageHeader
        title="Nuova Fattura"
        description="Crea una nuova fattura elettronica"
        className="mb-3"
      >
        <Button
          variant="falcon-default"
          size="sm"
          className="me-2"
          onClick={() => navigate('/billing/invoices/issued')}
        >
          <FontAwesomeIcon icon={faArrowLeft} className="me-1" />
          Torna alla lista
        </Button>
      </PageHeader>

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

      <Card className="mb-3">
        <FalconCardHeader title="Dati Fattura" light={false} />
        <Card.Body>
          <Tab.Container activeKey={activeTab} onSelect={(k) => setActiveTab(k || 'document')}>
            <Nav variant="tabs" className="mb-3">
              <Nav.Item>
                <Nav.Link eventKey="document">Documento</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="lines">Righe ({lines.length})</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="payment">Pagamento</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="options">Opzioni</Nav.Link>
              </Nav.Item>
            </Nav>

            <Tab.Content>
              {/* Document Tab */}
              <Tab.Pane eventKey="document">
                <Row>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Tipo Documento <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Select
                        value={documentType}
                        onChange={(e) => setDocumentType(e.target.value as DocumentType)}
                      >
                        {DOCUMENT_TYPES.map((dt) => (
                          <option key={dt.value} value={dt.value}>
                            {dt.label}
                          </option>
                        ))}
                      </Form.Select>
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Numero Fattura <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Control
                        type="text"
                        value={number}
                        onChange={(e) => setNumber(e.target.value)}
                        placeholder="es. 2026/001"
                      />
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Data <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Control
                        type="date"
                        value={date}
                        onChange={(e) => setDate(e.target.value)}
                      />
                    </Form.Group>
                  </Col>
                </Row>

                <Row>
                  <Col md={8}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Cliente <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Select
                        value={customerId}
                        onChange={(e) => setCustomerId(e.target.value)}
                      >
                        <option value="">Seleziona cliente...</option>
                        {customersData?.customers?.map((customer) => (
                          <option key={customer.id} value={customer.id}>
                            {customer.isCompany
                              ? customer.denomination
                              : `${customer.name} ${customer.surname}`}{' '}
                            - {customer.fiscalIdCode}
                          </option>
                        ))}
                      </Form.Select>
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    {selectedCustomer && (
                      <div className="mt-4 text-muted small">
                        <div>
                          <strong>SDI:</strong>{' '}
                          {selectedCustomer.codiceDestinatario || selectedCustomer.pecDestinatario || 'N/A'}
                        </div>
                        <div>
                          <strong>P.IVA:</strong> {selectedCustomer.fiscalIdCode}
                        </div>
                      </div>
                    )}
                  </Col>
                </Row>

                <Form.Group className="mb-3">
                  <Form.Label>Causale / Descrizione</Form.Label>
                  {causale.map((c, index) => (
                    <InputGroup className="mb-2" key={index}>
                      <Form.Control
                        type="text"
                        value={c}
                        onChange={(e) => handleCausaleChange(index, e.target.value)}
                        placeholder="es. Consulenza informatica mese di gennaio 2026"
                        maxLength={200}
                      />
                      {causale.length > 1 && (
                        <Button
                          variant="outline-danger"
                          onClick={() => handleRemoveCausale(index)}
                        >
                          <FontAwesomeIcon icon={faTrash} />
                        </Button>
                      )}
                    </InputGroup>
                  ))}
                  <Button variant="link" size="sm" onClick={handleAddCausale}>
                    <FontAwesomeIcon icon={faPlus} className="me-1" />
                    Aggiungi riga causale
                  </Button>
                </Form.Group>
              </Tab.Pane>

              {/* Lines Tab */}
              <Tab.Pane eventKey="lines">
                <div className="table-responsive">
                  <Table bordered hover size="sm">
                    <thead className="bg-light">
                      <tr>
                        <th style={{ width: '30%' }}>Descrizione *</th>
                        <th style={{ width: '8%' }}>Qtà *</th>
                        <th style={{ width: '10%' }}>U.M.</th>
                        <th style={{ width: '12%' }}>Prezzo Unit.</th>
                        <th style={{ width: '8%' }}>IVA %</th>
                        <th style={{ width: '15%' }}>Natura</th>
                        <th style={{ width: '12%' }}>Totale</th>
                        <th style={{ width: '5%' }}></th>
                      </tr>
                    </thead>
                    <tbody>
                      {lines.map((line, index) => {
                        const { totalPrice } = calculateLineTotals(line);
                        return (
                          <tr key={index}>
                            <td>
                              <Form.Control
                                size="sm"
                                type="text"
                                value={line.description}
                                onChange={(e) =>
                                  handleLineChange(index, 'description', e.target.value)
                                }
                                placeholder="Descrizione"
                              />
                            </td>
                            <td>
                              <Form.Control
                                size="sm"
                                type="number"
                                min="0"
                                step="0.01"
                                value={line.quantity}
                                onChange={(e) =>
                                  handleLineChange(index, 'quantity', parseFloat(e.target.value) || 0)
                                }
                              />
                            </td>
                            <td>
                              <Form.Select
                                size="sm"
                                value={line.unitOfMeasure || ''}
                                onChange={(e) =>
                                  handleLineChange(index, 'unitOfMeasure', e.target.value as UnitOfMeasure)
                                }
                              >
                                <option value="">-</option>
                                {UNITS_OF_MEASURE.map((um) => (
                                  <option key={um.value} value={um.value}>
                                    {um.value}
                                  </option>
                                ))}
                              </Form.Select>
                            </td>
                            <td>
                              <Form.Control
                                size="sm"
                                type="number"
                                min="0"
                                step="0.01"
                                value={line.unitPrice}
                                onChange={(e) =>
                                  handleLineChange(index, 'unitPrice', parseFloat(e.target.value) || 0)
                                }
                              />
                            </td>
                            <td>
                              <Form.Select
                                size="sm"
                                value={line.vatRate}
                                onChange={(e) =>
                                  handleLineChange(index, 'vatRate', parseFloat(e.target.value))
                                }
                              >
                                {VAT_RATES.map((rate) => (
                                  <option key={rate} value={rate}>
                                    {rate}%
                                  </option>
                                ))}
                              </Form.Select>
                            </td>
                            <td>
                              {line.vatRate === 0 ? (
                                <Form.Select
                                  size="sm"
                                  value={line.vatNature || ''}
                                  onChange={(e) =>
                                    handleLineChange(index, 'vatNature', e.target.value as VATNature || undefined)
                                  }
                                >
                                  <option value="">Seleziona...</option>
                                  {VAT_NATURES.map((n) => (
                                    <option key={n.value} value={n.value}>
                                      {n.value}
                                    </option>
                                  ))}
                                </Form.Select>
                              ) : (
                                <span className="text-muted">-</span>
                              )}
                            </td>
                            <td className="text-end">
                              <strong>
                                {totalPrice.toLocaleString('it-IT', {
                                  style: 'currency',
                                  currency: 'EUR',
                                })}
                              </strong>
                            </td>
                            <td>
                              <Button
                                variant="outline-danger"
                                size="sm"
                                onClick={() => handleRemoveLine(index)}
                                disabled={lines.length === 1}
                              >
                                <FontAwesomeIcon icon={faTrash} />
                              </Button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                    <tfoot>
                      <tr>
                        <td colSpan={8}>
                          <Button variant="falcon-primary" size="sm" onClick={handleAddLine}>
                            <FontAwesomeIcon icon={faPlus} className="me-1" />
                            Aggiungi Riga
                          </Button>
                        </td>
                      </tr>
                    </tfoot>
                  </Table>
                </div>

                {/* Totals */}
                <Row className="justify-content-end mt-3">
                  <Col md={4}>
                    <Table size="sm" className="border">
                      <tbody>
                        <tr>
                          <td>Imponibile</td>
                          <td className="text-end">
                            {totals.taxable.toLocaleString('it-IT', {
                              style: 'currency',
                              currency: 'EUR',
                            })}
                          </td>
                        </tr>
                        <tr>
                          <td>IVA</td>
                          <td className="text-end">
                            {totals.vat.toLocaleString('it-IT', {
                              style: 'currency',
                              currency: 'EUR',
                            })}
                          </td>
                        </tr>
                        <tr className="fw-bold">
                          <td>Totale</td>
                          <td className="text-end">
                            {totals.total.toLocaleString('it-IT', {
                              style: 'currency',
                              currency: 'EUR',
                            })}
                          </td>
                        </tr>
                      </tbody>
                    </Table>
                  </Col>
                </Row>
              </Tab.Pane>

              {/* Payment Tab */}
              <Tab.Pane eventKey="payment">
                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Condizione di Pagamento</Form.Label>
                      <Form.Select
                        value={paymentCondition}
                        onChange={(e) => setPaymentCondition(e.target.value as PaymentCondition)}
                      >
                        {PAYMENT_CONDITIONS.map((pc) => (
                          <option key={pc.value} value={pc.value}>
                            {pc.label}
                          </option>
                        ))}
                      </Form.Select>
                    </Form.Group>
                  </Col>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Metodo di Pagamento</Form.Label>
                      <Form.Select
                        value={paymentMethod}
                        onChange={(e) => setPaymentMethod(e.target.value as PaymentMethod)}
                      >
                        {PAYMENT_METHODS.map((pm) => (
                          <option key={pm.value} value={pm.value}>
                            {pm.label}
                          </option>
                        ))}
                      </Form.Select>
                    </Form.Group>
                  </Col>
                </Row>

                <Row>
                  <Col md={8}>
                    <Form.Group className="mb-3">
                      <Form.Label>IBAN</Form.Label>
                      <Form.Control
                        type="text"
                        value={paymentIban}
                        onChange={(e) => setPaymentIban(e.target.value.toUpperCase())}
                        placeholder="es. IT60X0542811101000000123456"
                        maxLength={34}
                      />
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>Scadenza Pagamento</Form.Label>
                      <Form.Control
                        type="date"
                        value={paymentDueDate}
                        onChange={(e) => setPaymentDueDate(e.target.value)}
                      />
                    </Form.Group>
                  </Col>
                </Row>
              </Tab.Pane>

              {/* Options Tab */}
              <Tab.Pane eventKey="options">
                <Form.Group className="mb-3">
                  <Form.Check
                    type="switch"
                    id="signatureEnabled"
                    label="Applica Firma Digitale"
                    checked={signatureEnabled}
                    onChange={(e) => setSignatureEnabled(e.target.checked)}
                  />
                  <Form.Text className="text-muted">
                    La fattura verrà firmata digitalmente prima dell'invio al SDI
                  </Form.Text>
                </Form.Group>

                <Form.Group className="mb-3">
                  <Form.Check
                    type="switch"
                    id="legalStorageEnabled"
                    label="Conservazione Sostitutiva"
                    checked={legalStorageEnabled}
                    onChange={(e) => setLegalStorageEnabled(e.target.checked)}
                  />
                  <Form.Text className="text-muted">
                    La fattura verrà conservata a norma di legge per 10 anni
                  </Form.Text>
                </Form.Group>

                <Form.Group className="mb-3">
                  <Form.Label>Note Interne</Form.Label>
                  <Form.Control
                    as="textarea"
                    rows={3}
                    value={internalNotes}
                    onChange={(e) => setInternalNotes(e.target.value)}
                    placeholder="Note visibili solo internamente (non inviate al SDI)"
                  />
                </Form.Group>
              </Tab.Pane>
            </Tab.Content>
          </Tab.Container>
        </Card.Body>
      </Card>

      {/* Action Buttons */}
      <Card>
        <Card.Body className="d-flex justify-content-between">
          <Button
            variant="falcon-default"
            onClick={() => navigate('/billing/invoices/issued')}
            disabled={isLoading}
          >
            Annulla
          </Button>
          <div>
            <Button
              variant="falcon-primary"
              className="me-2"
              onClick={handleSaveDraft}
              disabled={isLoading}
            >
              <FontAwesomeIcon icon={faSave} className="me-1" />
              {isCreating ? 'Salvataggio...' : 'Salva Bozza'}
            </Button>
            <Button
              variant="primary"
              onClick={handleSaveAndSend}
              disabled={isLoading}
            >
              <FontAwesomeIcon icon={faPaperPlane} className="me-1" />
              {isSending ? 'Invio...' : 'Salva e Invia al SDI'}
            </Button>
          </div>
        </Card.Body>
      </Card>
    </>
  );
};

export default NewIssuedInvoice;
