import React, { useState } from 'react';
import { Card, Button, Modal, Form, Alert } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { VehicleResponse, useUpdateVehicleMutation, useDeleteVehicleMutation } from 'store/api/vehicleApi';
import { useNavigate } from 'react-router';
import paths from 'routes/paths';
import { FaPowerOff, FaTrashAlt, FaCalendarCheck, FaDownload, FaQrcode } from 'react-icons/fa';

interface VehicleActionsProps {
  vehicle: VehicleResponse;
}

const VehicleActions: React.FC<VehicleActionsProps> = ({ vehicle }) => {
  const navigate = useNavigate();
  const [showActivationModal, setShowActivationModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showScheduleModal, setShowScheduleModal] = useState(false);
  const [scheduleDate, setScheduleDate] = useState('');

  const [updateVehicle, { isLoading: isUpdating }] = useUpdateVehicleMutation();
  const [deleteVehicle, { isLoading: isDeleting }] = useDeleteVehicleMutation();

  const handleToggleActivation = async () => {
    try {
      await updateVehicle({
        id: vehicle.id,
        data: { isActive: !vehicle.isActive }
      }).unwrap();
      setShowActivationModal(false);
    } catch (error) {
      console.error('Failed to update vehicle status:', error);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteVehicle(vehicle.id).unwrap();
      setShowDeleteModal(false);
      navigate(paths.fleetVehicles);
    } catch (error) {
      console.error('Failed to delete vehicle:', error);
    }
  };

  const handleScheduleRevision = async () => {
    if (!scheduleDate) return;

    try {
      await updateVehicle({
        id: vehicle.id,
        data: { revisioneProgrammata: new Date(scheduleDate).toISOString() }
      }).unwrap();
      setShowScheduleModal(false);
      setScheduleDate('');
    } catch (error) {
      console.error('Failed to schedule revision:', error);
    }
  };

  const handleExportPDF = () => {
    // In a real app, this would generate a PDF report
    console.log('Exporting vehicle report as PDF...');
  };

  const handlePrintQRCode = () => {
    // In a real app, this would generate and print a QR code
    console.log('Printing QR code for vehicle...');
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
              variant={vehicle.isActive ? 'warning' : 'success'}
              size="sm"
              onClick={() => setShowActivationModal(true)}
            >
              <FaPowerOff className="me-2" />
              {vehicle.isActive ? 'Disattiva Veicolo' : 'Attiva Veicolo'}
            </Button>

            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowScheduleModal(true)}
            >
              <FaCalendarCheck className="me-2" />
              Programma Revisione
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

            <hr />

            <Button
              variant="outline-danger"
              size="sm"
              onClick={() => setShowDeleteModal(true)}
            >
              <FaTrashAlt className="me-2" />
              Elimina Veicolo
            </Button>
          </div>
        </Card.Body>
      </Card>

      {/* Activation/Deactivation Modal */}
      <Modal show={showActivationModal} onHide={() => setShowActivationModal(false)} centered>
        <Modal.Header closeButton>
          <Modal.Title>
            {vehicle.isActive ? 'Disattiva Veicolo' : 'Attiva Veicolo'}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <p>
            Sei sicuro di voler {vehicle.isActive ? 'disattivare' : 'attivare'} il veicolo{' '}
            <strong>{vehicle.nome}</strong> (Targa: {vehicle.targa})?
          </p>
          {vehicle.isActive && (
            <Alert variant="warning">
              <FontAwesomeIcon icon="exclamation-triangle" className="me-2" />
              Il veicolo non sarà più disponibile per l'assegnazione ai conducenti.
            </Alert>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowActivationModal(false)}>
            Annulla
          </Button>
          <Button
            variant={vehicle.isActive ? 'warning' : 'success'}
            onClick={handleToggleActivation}
            disabled={isUpdating}
          >
            {isUpdating ? 'Attendere...' : vehicle.isActive ? 'Disattiva' : 'Attiva'}
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Delete Modal */}
      <Modal show={showDeleteModal} onHide={() => setShowDeleteModal(false)} centered>
        <Modal.Header closeButton>
          <Modal.Title>Elimina Veicolo</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Alert variant="danger">
            <FontAwesomeIcon icon="exclamation-triangle" className="me-2" />
            <strong>Attenzione!</strong> Questa azione è irreversibile.
          </Alert>
          <p>
            Sei sicuro di voler eliminare definitivamente il veicolo{' '}
            <strong>{vehicle.nome}</strong> (Targa: {vehicle.targa})?
          </p>
          <p className="text-muted small">
            Tutti i dati associati a questo veicolo, incluso lo storico manutenzione,
            verranno eliminati permanentemente.
          </p>
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
            {isDeleting ? 'Eliminazione...' : 'Elimina Definitivamente'}
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Schedule Revision Modal */}
      <Modal show={showScheduleModal} onHide={() => setShowScheduleModal(false)} centered>
        <Modal.Header closeButton>
          <Modal.Title>Programma Revisione</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <p>
            Seleziona la data per la prossima revisione del veicolo{' '}
            <strong>{vehicle.nome}</strong>:
          </p>
          <Form.Group className="mb-3">
            <Form.Label>Data Revisione Programmata</Form.Label>
            <Form.Control
              type="date"
              value={scheduleDate}
              onChange={(e) => setScheduleDate(e.target.value)}
              min={new Date().toISOString().split('T')[0]}
            />
          </Form.Group>
          {vehicle.scadenzaRevisione && (
            <Alert variant="info">
              <FontAwesomeIcon icon="info-circle" className="me-2" />
              Scadenza attuale: {new Date(vehicle.scadenzaRevisione).toLocaleDateString('it-IT')}
            </Alert>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowScheduleModal(false)}>
            Annulla
          </Button>
          <Button
            variant="primary"
            onClick={handleScheduleRevision}
            disabled={!scheduleDate || isUpdating}
          >
            {isUpdating ? 'Salvando...' : 'Programma'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default VehicleActions;