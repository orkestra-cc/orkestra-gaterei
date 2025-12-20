import React, { useState } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Collapse, Row, Col, Badge, Form } from 'react-bootstrap';
import {
  VehicleResponse,
  useUpdateVehicleMutation
} from 'store/api/vehicleApi';
import { FaTruck, FaTrailer, FaEdit, FaSave, FaTimes } from 'react-icons/fa';

interface VehicleProfileInfoProps {
  vehicle: VehicleResponse;
}

const VehicleProfileInfo: React.FC<VehicleProfileInfoProps> = ({ vehicle }) => {
  const [collapsed, setCollapsed] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [updateVehicle, { isLoading }] = useUpdateVehicleMutation();

  const [formData, setFormData] = useState({
    nome: vehicle.nome,
    targa: vehicle.targa,
    tipo: vehicle.tipo,
    luogo: vehicle.luogo || '',
    note: vehicle.note || '',
    scadenzaRevisione: vehicle.scadenzaRevisione
      ? new Date(vehicle.scadenzaRevisione).toISOString().split('T')[0]
      : '',
    revisioneProgrammata: vehicle.revisioneProgrammata
      ? new Date(vehicle.revisioneProgrammata).toISOString().split('T')[0]
      : '',
    insuranceExpiry: vehicle.insuranceExpiry
      ? new Date(vehicle.insuranceExpiry).toISOString().split('T')[0]
      : '',
    carTaxExpiry: vehicle.carTaxExpiry
      ? new Date(vehicle.carTaxExpiry).toISOString().split('T')[0]
      : ''
  });

  // Helper function to format date with time
  const formatDateTime = (dateString: string) => {
    return new Date(dateString).toLocaleString('it-IT', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  // Helper function to format date only
  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Non specificata';
    return new Date(dateString).toLocaleDateString('it-IT', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  };

  // Type labels
  const tipoLabels: Record<string, string> = {
    motrice: 'Motrice',
    rimorchio: 'Rimorchio',
    'semi-rimorchio': 'Semi-rimorchio',
    trattore: 'Trattore',
    semovente: 'Semovente'
  };

  const handleEdit = () => {
    setIsEditing(true);
    setCollapsed(false);
  };

  const handleCancel = () => {
    setIsEditing(false);
    // Reset form data to original values
    setFormData({
      nome: vehicle.nome,
      targa: vehicle.targa,
      tipo: vehicle.tipo,
      luogo: vehicle.luogo || '',
      note: vehicle.note || '',
      scadenzaRevisione: vehicle.scadenzaRevisione
        ? new Date(vehicle.scadenzaRevisione).toISOString().split('T')[0]
        : '',
      revisioneProgrammata: vehicle.revisioneProgrammata
        ? new Date(vehicle.revisioneProgrammata).toISOString().split('T')[0]
        : '',
      insuranceExpiry: vehicle.insuranceExpiry
        ? new Date(vehicle.insuranceExpiry).toISOString().split('T')[0]
        : '',
      carTaxExpiry: vehicle.carTaxExpiry
        ? new Date(vehicle.carTaxExpiry).toISOString().split('T')[0]
        : ''
    });
  };

  const handleSave = async () => {
    try {
      const dataToSubmit: any = {
        nome: formData.nome,
        targa: formData.targa.toUpperCase(),
        tipo: formData.tipo
      };

      // Add optional fields if they have values
      if (formData.luogo) dataToSubmit.luogo = formData.luogo;
      if (formData.note) dataToSubmit.note = formData.note;
      if (formData.scadenzaRevisione) {
        dataToSubmit.scadenzaRevisione = new Date(
          formData.scadenzaRevisione
        ).toISOString();
      }
      if (formData.revisioneProgrammata) {
        dataToSubmit.revisioneProgrammata = new Date(
          formData.revisioneProgrammata
        ).toISOString();
      }
      if (formData.insuranceExpiry) {
        dataToSubmit.insuranceExpiry = new Date(
          formData.insuranceExpiry
        ).toISOString();
      }
      if (formData.carTaxExpiry) {
        dataToSubmit.carTaxExpiry = new Date(
          formData.carTaxExpiry
        ).toISOString();
      }

      await updateVehicle({
        id: vehicle.id,
        data: dataToSubmit
      }).unwrap();

      setIsEditing(false);
    } catch (error) {
      console.error('Failed to update vehicle:', error);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<
      HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
    >
  ) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  return (
    <Card className="mb-3">
      <Card.Header className="bg-body-tertiary d-flex justify-content-between align-items-center">
        <h5 className="mb-0">Informazioni Veicolo</h5>
        <div>
          {!isEditing ? (
            <Button variant="falcon-default" size="sm" onClick={handleEdit}>
              <FaEdit className="me-1" /> Modifica
            </Button>
          ) : (
            <>
              <Button
                variant="success"
                size="sm"
                className="me-2"
                onClick={handleSave}
                disabled={isLoading}
              >
                <FaSave className="me-1" /> Salva
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleCancel}
                disabled={isLoading}
              >
                <FaTimes className="me-1" /> Annulla
              </Button>
            </>
          )}
        </div>
      </Card.Header>

      <Card.Body className="text-1000">
        <Row className="mb-3">
          <Col md={6}>
            <small className="text-700 d-block mb-1">Nome Veicolo</small>
            {isEditing ? (
              <Form.Control
                type="text"
                name="nome"
                value={formData.nome}
                onChange={handleChange}
                size="sm"
              />
            ) : (
              <div className="fw-semi-bold">
                {vehicle.tipo === 'motrice' || vehicle.tipo === 'trattore' || vehicle.tipo === 'semovente' ? (
                  <FaTruck className="me-2 text-primary" />
                ) : (
                  <FaTrailer className="me-2 text-primary" />
                )}
                {vehicle.nome}
              </div>
            )}
          </Col>
          <Col md={6}>
            <small className="text-700 d-block mb-1">Targa</small>
            {isEditing ? (
              <Form.Control
                type="text"
                name="targa"
                value={formData.targa}
                onChange={handleChange}
                size="sm"
              />
            ) : (
              <div className="fw-semi-bold">
                <FontAwesomeIcon icon="id-card" className="me-2 text-muted" />
                {vehicle.targa}
              </div>
            )}
          </Col>
        </Row>

        <Row className="mb-3">
          <Col md={6}>
            <small className="text-700 d-block mb-1">Tipo Veicolo</small>
            {isEditing ? (
              <Form.Select
                name="tipo"
                value={formData.tipo}
                onChange={handleChange}
                size="sm"
              >
                <option value="motrice">Motrice</option>
                <option value="rimorchio">Rimorchio</option>
                <option value="semi-rimorchio">Semi-rimorchio</option>
                <option value="trattore">Trattore</option>
                <option value="semovente">Semovente</option>
              </Form.Select>
            ) : (
              <Badge bg="soft-primary">
                {tipoLabels[vehicle.tipo] || vehicle.tipo}
              </Badge>
            )}
          </Col>
          <Col md={6}>
            <small className="text-700 d-block mb-1">Stato</small>
            <div>
              <Badge bg={vehicle.isActive ? 'soft-success' : 'soft-secondary'}>
                <span
                  className={`text-${vehicle.isActive ? 'success' : 'secondary'}`}
                >
                  {vehicle.isActive ? 'Attivo' : 'Inattivo'}
                </span>
              </Badge>
            </div>
          </Col>
        </Row>

        <Row className="mb-3">
          <Col md={6}>
            <small className="text-700 d-block mb-1">Posizione Attuale</small>
            {isEditing ? (
              <Form.Control
                type="text"
                name="luogo"
                value={formData.luogo}
                onChange={handleChange}
                size="sm"
                placeholder="Es. Deposito Calcinaia"
              />
            ) : (
              <div className="fw-semi-bold">
                <FontAwesomeIcon
                  icon="map-marker-alt"
                  className="me-2 text-muted"
                />
                {vehicle.luogo || 'Non specificata'}
              </div>
            )}
          </Col>
          <Col md={6}>
            <small className="text-700 d-block mb-1">Creato il</small>
            <div>
              <FontAwesomeIcon
                icon="calendar-plus"
                className="me-2 text-muted"
              />
              {formatDate(vehicle.createdAt)}
            </div>
          </Col>
        </Row>

        <hr className="my-3" />

        <h6 className="mb-3">Informazioni Revisione</h6>
        <Row className="mb-3">
          <Col md={6}>
            <small className="text-700 d-block mb-1">Scadenza Revisione</small>
            {isEditing ? (
              <Form.Control
                type="date"
                name="scadenzaRevisione"
                value={formData.scadenzaRevisione}
                onChange={handleChange}
                size="sm"
              />
            ) : (
              <div className="fw-semi-bold">
                <FontAwesomeIcon
                  icon="calendar-alt"
                  className="me-2 text-warning"
                />
                {formatDate(vehicle.scadenzaRevisione)}
              </div>
            )}
          </Col>
          <Col md={6}>
            <small className="text-700 d-block mb-1">
              Revisione Programmata
            </small>
            {isEditing ? (
              <Form.Control
                type="date"
                name="revisioneProgrammata"
                value={formData.revisioneProgrammata}
                onChange={handleChange}
                size="sm"
              />
            ) : (
              <div className="fw-semi-bold">
                <FontAwesomeIcon
                  icon="calendar-check"
                  className="me-2 text-success"
                />
                {formatDate(vehicle.revisioneProgrammata)}
              </div>
            )}
          </Col>
        </Row>

        <hr className="my-3" />

        <h6 className="mb-3">Informazioni Assicurazione e Bollo</h6>
        <Row className="mb-3">
          <Col md={6}>
            <small className="text-700 d-block mb-1">Scadenza Assicurazione</small>
            {isEditing ? (
              <Form.Control
                type="date"
                name="insuranceExpiry"
                value={formData.insuranceExpiry}
                onChange={handleChange}
                size="sm"
              />
            ) : (
              <div className="fw-semi-bold">
                <FontAwesomeIcon
                  icon="shield-alt"
                  className="me-2 text-info"
                />
                {formatDate(vehicle.insuranceExpiry)}
              </div>
            )}
          </Col>
          <Col md={6}>
            <small className="text-700 d-block mb-1">Scadenza Bollo</small>
            {isEditing ? (
              <Form.Control
                type="date"
                name="carTaxExpiry"
                value={formData.carTaxExpiry}
                onChange={handleChange}
                size="sm"
              />
            ) : (
              <div className="fw-semi-bold">
                <FontAwesomeIcon
                  icon="file-invoice-dollar"
                  className="me-2 text-danger"
                />
                {formatDate(vehicle.carTaxExpiry)}
              </div>
            )}
          </Col>
        </Row>

        <Button
          variant="link"
          size="sm"
          onClick={() => setCollapsed(!collapsed)}
          aria-controls="collapse-additional-info"
          aria-expanded={!collapsed}
          className="p-0"
        >
          {collapsed ? 'Mostra' : 'Nascondi'} dettagli aggiuntivi
          <FontAwesomeIcon
            icon="chevron-down"
            transform={collapsed ? 'rotate-180' : ''}
            className="ms-2"
          />
        </Button>

        <Collapse in={!collapsed}>
          <div id="collapse-additional-info">
            <hr className="my-3" />

            <Row className="mb-3">
              <Col xs={12}>
                <small className="text-700 d-block mb-1">Note</small>
                {isEditing ? (
                  <Form.Control
                    as="textarea"
                    rows={3}
                    name="note"
                    value={formData.note}
                    onChange={handleChange}
                    placeholder="Inserisci eventuali note sul veicolo..."
                  />
                ) : (
                  <div className="fw-semi-bold">
                    {vehicle.note || (
                      <span className="text-muted">
                        Nessuna nota disponibile
                      </span>
                    )}
                  </div>
                )}
              </Col>
            </Row>

            <Row>
              <Col md={6}>
                <small className="text-700 d-block mb-1">
                  Ultimo Aggiornamento
                </small>
                <div className="text-muted">
                  <FontAwesomeIcon icon="clock" className="me-2" />
                  {formatDateTime(vehicle.updatedAt)}
                </div>
              </Col>
              <Col md={6}>
                <small className="text-700 d-block mb-1">ID Veicolo</small>
                <div className="text-muted font-monospace small">
                  {vehicle.id}
                </div>
              </Col>
            </Row>
          </div>
        </Collapse>
      </Card.Body>
    </Card>
  );
};

export default VehicleProfileInfo;
