import React, { useState } from 'react';
import { Card, Button, Modal, Form, Alert } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { CraneResponse, useUpdateCraneMutation, useDeleteCraneMutation } from 'store/api/craneApi';
import { useNavigate } from 'react-router';
import paths from 'routes/paths';
import { FaPowerOff, FaTrashAlt, FaDownload, FaQrcode, FaClipboardCheck } from 'react-icons/fa';

interface CraneActionsProps {
  crane: CraneResponse;
}

const CraneActions: React.FC<CraneActionsProps> = ({ crane }) => {
  const navigate = useNavigate();
  const [showActivationModal, setShowActivationModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showScheduleModal, setShowScheduleModal] = useState(false);
  const [scheduleDate, setScheduleDate] = useState('');

  const [updateCrane, { isLoading: isUpdating }] = useUpdateCraneMutation();
  const [deleteCrane, { isLoading: isDeleting }] = useDeleteCraneMutation();

  const handleToggleActivation = async () => {
    try {
      await updateCrane({
        id: crane.id,
        data: { isActive: !crane.isActive }
      }).unwrap();
      setShowActivationModal(false);
    } catch (error) {
      console.error('Failed to update crane status:', error);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteCrane(crane.id).unwrap();
      setShowDeleteModal(false);
      navigate(paths.fleetCranes);
    } catch (error) {
      console.error('Failed to delete crane:', error);
    }
  };

  const handleScheduleVerification = async () => {
    if (!scheduleDate) return;

    try {
      await updateCrane({
        id: crane.id,
        data: { scadenzaVerifica: new Date(scheduleDate).toISOString() }
      }).unwrap();
      setShowScheduleModal(false);
      setScheduleDate('');
    } catch (error) {
      console.error('Failed to schedule verification:', error);
    }
  };

  const handleExportPDF = () => {
    // In a real app, this would generate a PDF report
    console.log('Exporting crane report as PDF...');
  };

  const handlePrintQRCode = () => {
    // In a real app, this would generate and print a QR code
    console.log('Printing QR code for crane...');
  };

  return (
    <>
      <Card className="mb-3">
        <Card.Header className="bg-body-tertiary">
          <h5 className="mb-0">Azioni Rapide</h5>
        </Card.Header>
        <Card.Body>
          <div className="d-grid gap-2">
            <Button
              variant={crane.isActive ? 'warning' : 'success'}
              size="sm"
              onClick={() => setShowActivationModal(true)}
            >
              <FaPowerOff className="me-2" />
              {crane.isActive ? 'Disattiva Gru' : 'Attiva Gru'}
            </Button>

            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowScheduleModal(true)}
            >
              <FaClipboardCheck className="me-2" />
              Programma Verifica
            </Button>

            <Button
              variant="outline-secondary"
              size="sm"
              onClick={handleExportPDF}
            >
              <FaDownload className="me-2" />
              Esporta Report PDF
            </Button>

            <Button
              variant="outline-secondary"
              size="sm"
              onClick={handlePrintQRCode}
            >
              <FaQrcode className="me-2" />
              Stampa QR Code
            </Button>

            <hr className="my-2" />

            <Button
              variant="danger"
              size="sm"
              onClick={() => setShowDeleteModal(true)}
            >
              <FaTrashAlt className="me-2" />
              Elimina Gru
            </Button>
          </div>
        </Card.Body>
      </Card>

      {/* Activation Modal */}
      <Modal show={showActivationModal} onHide={() => setShowActivationModal(false)} centered>
        <Modal.Header closeButton>
          <Modal.Title>
            {crane.isActive ? 'Disattiva Gru' : 'Attiva Gru'}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <p>
            Sei sicuro di voler {crane.isActive ? 'disattivare' : 'attivare'} la gru{' '}
            <strong>{crane.nome}</strong> (Matricola: {crane.matricola})?
          </p>
          {!crane.isActive && (
            <Alert variant="info">
              Attivando la gru, sarà disponibile per l'utilizzo e le verifiche.
            </Alert>
          )}
          {crane.isActive && (
            <Alert variant="warning">
              Disattivando la gru, non sarà più disponibile per l'utilizzo.
            </Alert>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowActivationModal(false)}>
            Annulla
          </Button>
          <Button
            variant={crane.isActive ? 'warning' : 'success'}
            onClick={handleToggleActivation}
            disabled={isUpdating}
          >
            {isUpdating ? 'Attendere...' : crane.isActive ? 'Disattiva' : 'Attiva'}
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Delete Modal */}
      <Modal show={showDeleteModal} onHide={() => setShowDeleteModal(false)} centered>
        <Modal.Header closeButton>
          <Modal.Title>Elimina Gru</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Alert variant="danger">
            <FontAwesomeIcon icon="exclamation-triangle" className="me-2" />
            Attenzione! Questa azione è irreversibile.
          </Alert>
          <p>
            Sei sicuro di voler eliminare definitivamente la gru{' '}
            <strong>{crane.nome}</strong> (Matricola: {crane.matricola})?
          </p>
          <p>Tutti i dati associati saranno persi.</p>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowDeleteModal(false)}>
            Annulla
          </Button>
          <Button
            variant="danger"
            onClick={handleDelete}
            disabled={isDeleting}
          >
            {isDeleting ? 'Eliminazione...' : 'Elimina Gru'}
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Schedule Verification Modal */}
      <Modal show={showScheduleModal} onHide={() => setShowScheduleModal(false)} centered>
        <Modal.Header closeButton>
          <Modal.Title>Programma Verifica</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <p>
            Programma la prossima verifica per la gru{' '}
            <strong>{crane.nome}</strong>.
          </p>
          <Form.Group>
            <Form.Label>Data Verifica</Form.Label>
            <Form.Control
              type="date"
              value={scheduleDate}
              onChange={(e) => setScheduleDate(e.target.value)}
              min={new Date().toISOString().split('T')[0]}
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowScheduleModal(false)}>
            Annulla
          </Button>
          <Button
            variant="primary"
            onClick={handleScheduleVerification}
            disabled={!scheduleDate || isUpdating}
          >
            {isUpdating ? 'Salvando...' : 'Programma'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default CraneActions;