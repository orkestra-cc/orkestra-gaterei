import React, { useState } from 'react';
import { Modal, Button, Form, Row, Col, Alert } from 'react-bootstrap';
import FalconCloseButton from 'components/common/FalconCloseButton';
import { useCreateCraneMutation } from 'store/api/craneApi';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

interface AddCraneModalProps {
  show: boolean;
  onHide: () => void;
}

const AddCraneModal: React.FC<AddCraneModalProps> = ({ show, onHide }) => {
  const [createCrane, { isLoading, error }] = useCreateCraneMutation();
  const [formData, setFormData] = useState({
    nome: '',
    tipo: '',
    matricola: '',
    verificareSuMezzo: '',
    scadenzaVerifica: '',
    note: ''
  });

  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
    // Clear validation error for this field
    if (validationErrors[name]) {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
  };

  const validateForm = () => {
    const errors: Record<string, string> = {};

    if (!formData.nome.trim()) {
      errors.nome = 'Il nome della gru è obbligatorio';
    }

    if (!formData.tipo.trim()) {
      errors.tipo = 'Il tipo di gru è obbligatorio';
    }

    if (!formData.matricola.trim()) {
      errors.matricola = 'La matricola è obbligatoria';
    }

    // Validate date if provided
    if (formData.scadenzaVerifica) {
      const date = new Date(formData.scadenzaVerifica);
      if (isNaN(date.getTime())) {
        errors.scadenzaVerifica = 'Data non valida';
      }
    }

    // Validate mezzo targa format if provided
    if (formData.verificareSuMezzo && !/^[A-Z0-9]+$/i.test(formData.verificareSuMezzo.replace(/\s/g, ''))) {
      errors.verificareSuMezzo = 'La targa del mezzo deve contenere solo lettere e numeri';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    try {
      // Prepare data for submission
      const dataToSubmit: any = {
        nome: formData.nome.trim(),
        tipo: formData.tipo.trim(),
        matricola: formData.matricola.trim()
      };

      // Add optional fields if they have values
      if (formData.verificareSuMezzo.trim()) {
        dataToSubmit.verificareSuMezzo = formData.verificareSuMezzo.trim().toUpperCase();
      }
      if (formData.scadenzaVerifica) {
        dataToSubmit.scadenzaVerifica = new Date(formData.scadenzaVerifica).toISOString();
      }
      if (formData.note.trim()) {
        dataToSubmit.note = formData.note.trim();
      }

      await createCrane(dataToSubmit).unwrap();

      // Reset form and close modal on success
      setFormData({
        nome: '',
        tipo: '',
        matricola: '',
        verificareSuMezzo: '',
        scadenzaVerifica: '',
        note: ''
      });
      setValidationErrors({});
      onHide();
    } catch (err) {
      console.error('Failed to create crane:', err);
    }
  };

  const handleClose = () => {
    // Reset form when closing
    setFormData({
      nome: '',
      tipo: '',
      matricola: '',
      verificareSuMezzo: '',
      scadenzaVerifica: '',
      note: ''
    });
    setValidationErrors({});
    onHide();
  };

  const craneTypes = [
    'Autogrù',
    'Gru a torre',
    'Gru mobile',
    'Gru fissa',
    'Gru semovente',
    'Gru su camion',
    'Gru cingolata',
    'Altro'
  ];

  return (
    <Modal show={show} onHide={handleClose} size="lg" centered>
      <Form onSubmit={handleSubmit}>
        <Modal.Header>
          <Modal.Title>
            <FontAwesomeIcon icon="plus" className="me-2" />
            Aggiungi Nuova Gru
          </Modal.Title>
          <FalconCloseButton onClick={handleClose} />
        </Modal.Header>
        <Modal.Body>
          {error && (
            <Alert variant="danger" className="mb-3">
              <FontAwesomeIcon icon="exclamation-triangle" className="me-2" />
              Si è verificato un errore durante la creazione della gru. Riprova.
            </Alert>
          )}

          <Row className="g-3">
            <Col md={6}>
              <Form.Group>
                <Form.Label>
                  Nome Gru <span className="text-danger">*</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  name="nome"
                  value={formData.nome}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.nome}
                  placeholder="Es. Gru Mobile 01"
                  required
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.nome}
                </Form.Control.Feedback>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>
                  Tipo <span className="text-danger">*</span>
                </Form.Label>
                <Form.Select
                  name="tipo"
                  value={formData.tipo}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.tipo}
                  required
                >
                  <option value="">Seleziona tipo...</option>
                  {craneTypes.map(type => (
                    <option key={type} value={type}>{type}</option>
                  ))}
                </Form.Select>
                <Form.Control.Feedback type="invalid">
                  {validationErrors.tipo}
                </Form.Control.Feedback>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>
                  Matricola <span className="text-danger">*</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  name="matricola"
                  value={formData.matricola}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.matricola}
                  placeholder="Inserisci matricola"
                  required
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.matricola}
                </Form.Control.Feedback>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>Targa Mezzo Associato</Form.Label>
                <Form.Control
                  type="text"
                  name="verificareSuMezzo"
                  value={formData.verificareSuMezzo}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.verificareSuMezzo}
                  placeholder="Es. AA123BB"
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.verificareSuMezzo}
                </Form.Control.Feedback>
                <Form.Text className="text-muted">
                  Inserire la targa del mezzo su cui è montata la gru
                </Form.Text>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>Scadenza Verifica</Form.Label>
                <Form.Control
                  type="date"
                  name="scadenzaVerifica"
                  value={formData.scadenzaVerifica}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.scadenzaVerifica}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.scadenzaVerifica}
                </Form.Control.Feedback>
                <Form.Text className="text-muted">
                  Data di scadenza della verifica periodica
                </Form.Text>
              </Form.Group>
            </Col>

            <Col xs={12}>
              <Form.Group>
                <Form.Label>Note</Form.Label>
                <Form.Control
                  as="textarea"
                  rows={3}
                  name="note"
                  value={formData.note}
                  onChange={handleChange}
                  placeholder="Inserisci eventuali note sulla gru..."
                />
              </Form.Group>
            </Col>
          </Row>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={handleClose} disabled={isLoading}>
            Annulla
          </Button>
          <Button variant="primary" type="submit" disabled={isLoading}>
            {isLoading ? (
              <>
                <FontAwesomeIcon icon="spinner" spin className="me-2" />
                Creazione...
              </>
            ) : (
              <>
                <FontAwesomeIcon icon="plus" className="me-2" />
                Crea Gru
              </>
            )}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default AddCraneModal;