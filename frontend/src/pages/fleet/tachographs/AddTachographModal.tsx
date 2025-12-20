import React, { useState } from 'react';
import { Modal, Button, Form, Row, Col, Alert } from 'react-bootstrap';
import FalconCloseButton from 'components/common/FalconCloseButton';
import { useCreateTachographMutation } from 'store/api/tachographApi';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

interface AddTachographModalProps {
  show: boolean;
  onHide: () => void;
}

const AddTachographModal: React.FC<AddTachographModalProps> = ({ show, onHide }) => {
  const [createTachograph, { isLoading, error }] = useCreateTachographMutation();
  const [formData, setFormData] = useState({
    nome: '',
    targa: '',
    luogo: '',
    note: '',
    scadenzaRevisione: '',
    revisioneProgrammata: ''
  });

  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
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

    // Nome validation (required, 1-100 chars)
    if (!formData.nome.trim()) {
      errors.nome = 'Il nome del tachigrafo è obbligatorio';
    } else if (formData.nome.trim().length > 100) {
      errors.nome = 'Il nome deve essere massimo 100 caratteri';
    }

    // Targa validation (required, 1-20 chars)
    if (!formData.targa.trim()) {
      errors.targa = 'La targa è obbligatoria';
    } else if (formData.targa.trim().length > 20) {
      errors.targa = 'La targa deve essere massimo 20 caratteri';
    } else if (!/^[A-Z0-9]+$/i.test(formData.targa.replace(/\s/g, ''))) {
      errors.targa = 'La targa deve contenere solo lettere e numeri';
    }

    // Luogo validation (optional, max 200 chars)
    if (formData.luogo.trim().length > 200) {
      errors.luogo = 'La posizione deve essere massimo 200 caratteri';
    }

    // Note validation (optional, max 500 chars)
    if (formData.note.trim().length > 500) {
      errors.note = 'Le note devono essere massimo 500 caratteri';
    }

    // Validate dates if provided
    if (formData.scadenzaRevisione) {
      const date = new Date(formData.scadenzaRevisione);
      if (isNaN(date.getTime())) {
        errors.scadenzaRevisione = 'Data non valida';
      }
    }

    if (formData.revisioneProgrammata) {
      const date = new Date(formData.revisioneProgrammata);
      if (isNaN(date.getTime())) {
        errors.revisioneProgrammata = 'Data non valida';
      }
      // Check if programmed revision is before expiry
      if (formData.scadenzaRevisione && !errors.scadenzaRevisione) {
        const expiryDate = new Date(formData.scadenzaRevisione);
        if (date > expiryDate) {
          errors.revisioneProgrammata = 'La revisione programmata non può essere dopo la scadenza';
        }
      }
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
        targa: formData.targa.trim().toUpperCase()
      };

      // Add optional fields if they have values
      if (formData.luogo.trim()) {
        dataToSubmit.luogo = formData.luogo.trim();
      }
      if (formData.note.trim()) {
        dataToSubmit.note = formData.note.trim();
      }
      if (formData.scadenzaRevisione) {
        dataToSubmit.scadenzaRevisione = new Date(formData.scadenzaRevisione).toISOString();
      }
      if (formData.revisioneProgrammata) {
        dataToSubmit.revisioneProgrammata = new Date(formData.revisioneProgrammata).toISOString();
      }

      await createTachograph(dataToSubmit).unwrap();

      // Reset form and close modal on success
      setFormData({
        nome: '',
        targa: '',
        luogo: '',
        note: '',
        scadenzaRevisione: '',
        revisioneProgrammata: ''
      });
      setValidationErrors({});
      onHide();
    } catch (err: any) {
      console.error('Failed to create tachograph:', err);
      // Handle specific error cases
      if (err?.data?.code === 409 || err?.status === 409) {
        setValidationErrors({ targa: 'Questa targa è già registrata nel sistema' });
      }
    }
  };

  const handleClose = () => {
    setFormData({
      nome: '',
      targa: '',
      luogo: '',
      note: '',
      scadenzaRevisione: '',
      revisioneProgrammata: ''
    });
    setValidationErrors({});
    onHide();
  };

  return (
    <Modal show={show} onHide={handleClose} size="lg" centered>
      <Form onSubmit={handleSubmit}>
        <Modal.Header>
          <Modal.Title>Aggiungi Nuovo Tachigrafo</Modal.Title>
          <FalconCloseButton onClick={handleClose} />
        </Modal.Header>
        <Modal.Body>
          {error && (
            <Alert variant="danger" dismissible onClose={() => {}}>
              <FontAwesomeIcon icon="exclamation-triangle" className="me-2" />
              Si è verificato un errore durante la creazione del tachigrafo.
            </Alert>
          )}

          <Row className="g-3">
            <Col md={6}>
              <Form.Group>
                <Form.Label>
                  Nome <span className="text-danger">*</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  name="nome"
                  value={formData.nome}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.nome}
                  placeholder="Es. Tachigrafo 001"
                  maxLength={100}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.nome}
                </Form.Control.Feedback>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>
                  Targa <span className="text-danger">*</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  name="targa"
                  value={formData.targa}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.targa}
                  placeholder="Es. AB123CD"
                  maxLength={20}
                  style={{ textTransform: 'uppercase' }}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.targa}
                </Form.Control.Feedback>
                <Form.Text className="text-muted">
                  La targa verrà salvata in maiuscolo
                </Form.Text>
              </Form.Group>
            </Col>

            <Col md={12}>
              <Form.Group>
                <Form.Label>Posizione</Form.Label>
                <Form.Control
                  type="text"
                  name="luogo"
                  value={formData.luogo}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.luogo}
                  placeholder="Es. Deposito principale, Via Roma 123"
                  maxLength={200}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.luogo}
                </Form.Control.Feedback>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>Scadenza Revisione</Form.Label>
                <Form.Control
                  type="date"
                  name="scadenzaRevisione"
                  value={formData.scadenzaRevisione}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.scadenzaRevisione}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.scadenzaRevisione}
                </Form.Control.Feedback>
              </Form.Group>
            </Col>

            <Col md={6}>
              <Form.Group>
                <Form.Label>Revisione Programmata</Form.Label>
                <Form.Control
                  type="date"
                  name="revisioneProgrammata"
                  value={formData.revisioneProgrammata}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.revisioneProgrammata}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.revisioneProgrammata}
                </Form.Control.Feedback>
                <Form.Text className="text-muted">
                  Data prevista per la prossima revisione
                </Form.Text>
              </Form.Group>
            </Col>

            <Col md={12}>
              <Form.Group>
                <Form.Label>Note</Form.Label>
                <Form.Control
                  as="textarea"
                  rows={3}
                  name="note"
                  value={formData.note}
                  onChange={handleChange}
                  isInvalid={!!validationErrors.note}
                  placeholder="Aggiungi note o informazioni aggiuntive..."
                  maxLength={500}
                />
                <Form.Control.Feedback type="invalid">
                  {validationErrors.note}
                </Form.Control.Feedback>
                <Form.Text className="text-muted">
                  {formData.note.length}/500 caratteri
                </Form.Text>
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
                Aggiungi Tachigrafo
              </>
            )}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default AddTachographModal;