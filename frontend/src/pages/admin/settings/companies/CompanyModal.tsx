import React, { useState, useEffect } from 'react';
import { Modal, Button, Form, Alert, Tab, Nav, Row, Col } from 'react-bootstrap';
import {
  useCreateCompanyMutation,
  useUpdateCompanyMutation,
} from 'store/api/billingApi';
import type { Company, CreateCompanyInput, UpdateCompanyInput, RegimeFiscale } from 'types/billing';
import { REGIME_FISCALE_LABELS, REGIME_FISCALE_OPTIONS } from 'types/billing';
import FalconCloseButton from 'components/common/FalconCloseButton';

interface CompanyModalProps {
  show: boolean;
  onHide: () => void;
  company?: Company | null;
  onSuccess?: () => void;
}

const CompanyModal: React.FC<CompanyModalProps> = ({
  show,
  onHide,
  company,
  onSuccess
}) => {
  const isEditMode = !!company;
  const [createCompany, { isLoading: isCreating }] = useCreateCompanyMutation();
  const [updateCompany, { isLoading: isUpdating }] = useUpdateCompanyMutation();
  const isLoading = isCreating || isUpdating;

  const [error, setError] = useState<string>('');
  const [activeTab, setActiveTab] = useState<string>('general');

  const initialFormData: CreateCompanyInput = {
    fiscalIdCountry: 'IT',
    fiscalIdCode: '',
    codiceFiscale: '',
    denomination: '',
    regimeFiscale: 'RF01',
    address: '',
    numeroCivico: '',
    city: '',
    province: '',
    postalCode: '',
    country: 'IT',
    // REA
    reaOffice: '',
    reaNumber: '',
    capitaleSociale: undefined,
    socioUnico: undefined,
    statoLiquidazione: 'LN',
    // Contacts
    email: '',
    pec: '',
    phone: '',
    // Bank
    iban: '',
    bic: '',
    abi: '',
    cab: '',
    beneficiario: '',
    istitutoFinanziario: '',
    notes: '',
  };

  const [formData, setFormData] = useState<CreateCompanyInput>(initialFormData);

  // Populate form when editing
  useEffect(() => {
    if (company && show) {
      setFormData({
        fiscalIdCountry: company.fiscalIdCountry || 'IT',
        fiscalIdCode: company.fiscalIdCode || '',
        codiceFiscale: company.codiceFiscale || '',
        denomination: company.denomination || '',
        regimeFiscale: company.regimeFiscale || 'RF01',
        address: company.address || '',
        numeroCivico: company.numeroCivico || '',
        city: company.city || '',
        province: company.province || '',
        postalCode: company.postalCode || '',
        country: company.country || 'IT',
        // REA
        reaOffice: company.reaOffice || '',
        reaNumber: company.reaNumber || '',
        capitaleSociale: company.capitaleSociale,
        socioUnico: company.socioUnico,
        statoLiquidazione: company.statoLiquidazione || 'LN',
        // Contacts
        email: company.email || '',
        pec: company.pec || '',
        phone: company.phone || '',
        // Bank
        iban: company.iban || '',
        bic: company.bic || '',
        abi: company.abi || '',
        cab: company.cab || '',
        beneficiario: company.beneficiario || '',
        istitutoFinanziario: company.istitutoFinanziario || '',
        notes: company.notes || '',
      });
    }
  }, [company, show]);

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value, type } = e.target;
    const checked = (e.target as HTMLInputElement).checked;

    setFormData((prev: CreateCompanyInput) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
    setError('');
  };

  const handleNumberChange = (
    e: React.ChangeEvent<HTMLInputElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev: CreateCompanyInput) => ({
      ...prev,
      [name]: value ? parseFloat(value) : undefined
    }));
    setError('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // Validation
    if (!formData.fiscalIdCode.trim()) {
      setError('La Partita IVA è obbligatoria');
      setActiveTab('general');
      return;
    }

    if (!formData.denomination?.trim()) {
      setError('La Ragione Sociale è obbligatoria');
      setActiveTab('general');
      return;
    }

    if (!formData.address.trim() || !formData.city.trim() || !formData.postalCode.trim()) {
      setError('Indirizzo, Città e CAP sono obbligatori');
      setActiveTab('address');
      return;
    }

    try {
      if (isEditMode && company) {
        const updateData: UpdateCompanyInput = {
          denomination: formData.denomination,
          regimeFiscale: formData.regimeFiscale,
          address: formData.address,
          numeroCivico: formData.numeroCivico,
          city: formData.city,
          province: formData.province,
          postalCode: formData.postalCode,
          country: formData.country,
          // REA
          reaOffice: formData.reaOffice,
          reaNumber: formData.reaNumber,
          capitaleSociale: formData.capitaleSociale,
          socioUnico: formData.socioUnico,
          statoLiquidazione: formData.statoLiquidazione,
          // Contacts
          email: formData.email,
          pec: formData.pec,
          phone: formData.phone,
          // Bank
          iban: formData.iban,
          bic: formData.bic,
          abi: formData.abi,
          cab: formData.cab,
          beneficiario: formData.beneficiario,
          istitutoFinanziario: formData.istitutoFinanziario,
          notes: formData.notes,
        };
        await updateCompany({ id: company.id, data: updateData }).unwrap();
      } else {
        await createCompany(formData).unwrap();
      }

      handleClose();
      if (onSuccess) onSuccess();
    } catch (err: any) {
      setError(err?.data?.message || `Errore durante ${isEditMode ? 'il salvataggio' : 'la creazione'} dell'azienda`);
    }
  };

  const handleClose = () => {
    setFormData(initialFormData);
    setError('');
    setActiveTab('general');
    onHide();
  };

  return (
    <Modal show={show} onHide={handleClose} centered size="lg">
      <Modal.Header>
        <Modal.Title>
          {isEditMode ? 'Modifica Azienda' : 'Nuova Azienda'}
        </Modal.Title>
        <FalconCloseButton onClick={handleClose} />
      </Modal.Header>
      <Form onSubmit={handleSubmit}>
        <Modal.Body>
          {error && (
            <Alert variant="danger" dismissible onClose={() => setError('')}>
              {error}
            </Alert>
          )}

          <Tab.Container activeKey={activeTab} onSelect={(k) => setActiveTab(k || 'general')}>
            <Nav variant="tabs" className="mb-3">
              <Nav.Item>
                <Nav.Link eventKey="general">Dati Generali</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="address">Indirizzo</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="rea">REA</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="contact">Contatti</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="bank">Banca</Nav.Link>
              </Nav.Item>
            </Nav>

            <Tab.Content>
              {/* General Tab */}
              <Tab.Pane eventKey="general">
                <Form.Group className="mb-3">
                  <Form.Label>
                    Ragione Sociale <span className="text-danger">*</span>
                  </Form.Label>
                  <Form.Control
                    type="text"
                    name="denomination"
                    value={formData.denomination}
                    onChange={handleChange}
                    placeholder="es. Acme S.r.l."
                    required
                  />
                </Form.Group>

                <Row>
                  <Col md={2}>
                    <Form.Group className="mb-3">
                      <Form.Label>Paese</Form.Label>
                      <Form.Control
                        type="text"
                        name="fiscalIdCountry"
                        value={formData.fiscalIdCountry}
                        onChange={handleChange}
                        maxLength={2}
                        disabled={isEditMode}
                      />
                    </Form.Group>
                  </Col>
                  <Col md={5}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Partita IVA <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Control
                        type="text"
                        name="fiscalIdCode"
                        value={formData.fiscalIdCode}
                        onChange={handleChange}
                        placeholder="es. 12345678901"
                        maxLength={11}
                        disabled={isEditMode}
                        required
                      />
                    </Form.Group>
                  </Col>
                  <Col md={5}>
                    <Form.Group className="mb-3">
                      <Form.Label>Codice Fiscale</Form.Label>
                      <Form.Control
                        type="text"
                        name="codiceFiscale"
                        value={formData.codiceFiscale}
                        onChange={handleChange}
                        placeholder="es. RSSMRA80A01H501Z"
                        maxLength={16}
                        disabled={isEditMode}
                      />
                      <Form.Text className="text-muted">
                        Solo se diverso dalla P.IVA
                      </Form.Text>
                    </Form.Group>
                  </Col>
                </Row>

                <Form.Group className="mb-3">
                  <Form.Label>
                    Regime Fiscale <span className="text-danger">*</span>
                  </Form.Label>
                  <Form.Select
                    name="regimeFiscale"
                    value={formData.regimeFiscale}
                    onChange={handleChange}
                    required
                  >
                    {REGIME_FISCALE_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </Form.Select>
                </Form.Group>
              </Tab.Pane>

              {/* Address Tab */}
              <Tab.Pane eventKey="address">
                <Row>
                  <Col md={9}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Indirizzo <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Control
                        type="text"
                        name="address"
                        value={formData.address}
                        onChange={handleChange}
                        placeholder="es. Via Roma"
                        required
                      />
                    </Form.Group>
                  </Col>
                  <Col md={3}>
                    <Form.Group className="mb-3">
                      <Form.Label>N. Civico</Form.Label>
                      <Form.Control
                        type="text"
                        name="numeroCivico"
                        value={formData.numeroCivico}
                        onChange={handleChange}
                        placeholder="123"
                        maxLength={10}
                      />
                    </Form.Group>
                  </Col>
                </Row>

                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        Città <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Control
                        type="text"
                        name="city"
                        value={formData.city}
                        onChange={handleChange}
                        placeholder="es. Roma"
                        required
                      />
                    </Form.Group>
                  </Col>
                  <Col md={2}>
                    <Form.Group className="mb-3">
                      <Form.Label>Prov.</Form.Label>
                      <Form.Control
                        type="text"
                        name="province"
                        value={formData.province}
                        onChange={handleChange}
                        placeholder="RM"
                        maxLength={2}
                      />
                    </Form.Group>
                  </Col>
                  <Col md={2}>
                    <Form.Group className="mb-3">
                      <Form.Label>
                        CAP <span className="text-danger">*</span>
                      </Form.Label>
                      <Form.Control
                        type="text"
                        name="postalCode"
                        value={formData.postalCode}
                        onChange={handleChange}
                        placeholder="00100"
                        maxLength={5}
                        required
                      />
                    </Form.Group>
                  </Col>
                  <Col md={2}>
                    <Form.Group className="mb-3">
                      <Form.Label>Nazione</Form.Label>
                      <Form.Control
                        type="text"
                        name="country"
                        value={formData.country}
                        onChange={handleChange}
                        maxLength={2}
                      />
                    </Form.Group>
                  </Col>
                </Row>
              </Tab.Pane>

              {/* REA Tab */}
              <Tab.Pane eventKey="rea">
                <Row>
                  <Col md={3}>
                    <Form.Group className="mb-3">
                      <Form.Label>Ufficio REA</Form.Label>
                      <Form.Control
                        type="text"
                        name="reaOffice"
                        value={formData.reaOffice}
                        onChange={handleChange}
                        placeholder="es. RM"
                        maxLength={2}
                      />
                      <Form.Text className="text-muted">
                        Sigla provincia
                      </Form.Text>
                    </Form.Group>
                  </Col>
                  <Col md={5}>
                    <Form.Group className="mb-3">
                      <Form.Label>Numero REA</Form.Label>
                      <Form.Control
                        type="text"
                        name="reaNumber"
                        value={formData.reaNumber}
                        onChange={handleChange}
                        placeholder="es. 123456"
                        maxLength={20}
                      />
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>Capitale Sociale</Form.Label>
                      <Form.Control
                        type="number"
                        name="capitaleSociale"
                        value={formData.capitaleSociale || ''}
                        onChange={handleNumberChange}
                        placeholder="es. 10000.00"
                        step="0.01"
                        min="0"
                      />
                    </Form.Group>
                  </Col>
                </Row>

                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Socio Unico</Form.Label>
                      <Form.Select
                        name="socioUnico"
                        value={formData.socioUnico || ''}
                        onChange={handleChange}
                      >
                        <option value="">Non specificato</option>
                        <option value="SU">SU - Socio Unico</option>
                        <option value="SM">SM - Più Soci</option>
                      </Form.Select>
                    </Form.Group>
                  </Col>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Stato Liquidazione</Form.Label>
                      <Form.Select
                        name="statoLiquidazione"
                        value={formData.statoLiquidazione || 'LN'}
                        onChange={handleChange}
                      >
                        <option value="LN">LN - Non in liquidazione</option>
                        <option value="LS">LS - In liquidazione</option>
                      </Form.Select>
                    </Form.Group>
                  </Col>
                </Row>

                <Alert variant="info" className="mt-3">
                  <small>
                    I dati REA (Repertorio Economico Amministrativo) sono obbligatori
                    per le società di capitali e cooperative iscritte al Registro Imprese.
                  </small>
                </Alert>
              </Tab.Pane>

              {/* Contact Tab */}
              <Tab.Pane eventKey="contact">
                <Form.Group className="mb-3">
                  <Form.Label>Email</Form.Label>
                  <Form.Control
                    type="email"
                    name="email"
                    value={formData.email}
                    onChange={handleChange}
                    placeholder="es. info@azienda.it"
                  />
                </Form.Group>

                <Form.Group className="mb-3">
                  <Form.Label>PEC</Form.Label>
                  <Form.Control
                    type="email"
                    name="pec"
                    value={formData.pec}
                    onChange={handleChange}
                    placeholder="es. azienda@pec.it"
                  />
                  <Form.Text className="text-muted">
                    Posta Elettronica Certificata
                  </Form.Text>
                </Form.Group>

                <Form.Group className="mb-3">
                  <Form.Label>Telefono</Form.Label>
                  <Form.Control
                    type="tel"
                    name="phone"
                    value={formData.phone}
                    onChange={handleChange}
                    placeholder="es. +39 06 1234567"
                  />
                </Form.Group>

                <Form.Group className="mb-3">
                  <Form.Label>Note</Form.Label>
                  <Form.Control
                    as="textarea"
                    rows={3}
                    name="notes"
                    value={formData.notes}
                    onChange={handleChange}
                    placeholder="Note interne..."
                  />
                </Form.Group>
              </Tab.Pane>

              {/* Bank Tab */}
              <Tab.Pane eventKey="bank">
                <Form.Group className="mb-3">
                  <Form.Label>IBAN</Form.Label>
                  <Form.Control
                    type="text"
                    name="iban"
                    value={formData.iban}
                    onChange={handleChange}
                    placeholder="es. IT60X0542811101000000123456"
                    maxLength={34}
                  />
                </Form.Group>

                <Row>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>BIC/SWIFT</Form.Label>
                      <Form.Control
                        type="text"
                        name="bic"
                        value={formData.bic}
                        onChange={handleChange}
                        placeholder="es. BLOPIT22"
                        maxLength={11}
                      />
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>ABI</Form.Label>
                      <Form.Control
                        type="text"
                        name="abi"
                        value={formData.abi}
                        onChange={handleChange}
                        placeholder="es. 05428"
                        maxLength={5}
                      />
                    </Form.Group>
                  </Col>
                  <Col md={4}>
                    <Form.Group className="mb-3">
                      <Form.Label>CAB</Form.Label>
                      <Form.Control
                        type="text"
                        name="cab"
                        value={formData.cab}
                        onChange={handleChange}
                        placeholder="es. 11101"
                        maxLength={5}
                      />
                    </Form.Group>
                  </Col>
                </Row>

                <Form.Group className="mb-3">
                  <Form.Label>Beneficiario</Form.Label>
                  <Form.Control
                    type="text"
                    name="beneficiario"
                    value={formData.beneficiario}
                    onChange={handleChange}
                    placeholder="es. Acme S.r.l."
                  />
                  <Form.Text className="text-muted">
                    Intestatario del conto corrente
                  </Form.Text>
                </Form.Group>

                <Form.Group className="mb-3">
                  <Form.Label>Istituto Finanziario</Form.Label>
                  <Form.Control
                    type="text"
                    name="istitutoFinanziario"
                    value={formData.istitutoFinanziario}
                    onChange={handleChange}
                    placeholder="es. Banca XYZ S.p.A."
                  />
                </Form.Group>
              </Tab.Pane>
            </Tab.Content>
          </Tab.Container>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={handleClose} disabled={isLoading}>
            Annulla
          </Button>
          <Button variant="primary" type="submit" disabled={isLoading}>
            {isLoading
              ? (isEditMode ? 'Salvataggio...' : 'Creazione...')
              : (isEditMode ? 'Salva Modifiche' : 'Crea Azienda')}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default CompanyModal;
