import React, { useState } from 'react';
import { Card, Button, Modal } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { TachographResponse, useUpdateTachographMutation, useDeleteTachographMutation } from 'store/api/tachographApi';
import { useNavigate } from 'react-router';
import paths from 'routes/paths';
import FalconCloseButton from 'components/common/FalconCloseButton';

interface TachographActionsProps {
  tachograph: TachographResponse;
}

const TachographActions: React.FC<TachographActionsProps> = ({ tachograph }) => {
  const navigate = useNavigate();
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showActivationModal, setShowActivationModal] = useState(false);
  const [updateTachograph, { isLoading: isUpdating }] = useUpdateTachographMutation();
  const [deleteTachograph, { isLoading: isDeleting }] = useDeleteTachographMutation();

  const handleToggleActivation = async () => {
    try {
      await updateTachograph({
        id: tachograph.id,
        data: { isActive: !tachograph.isActive }
      }).unwrap();
      setShowActivationModal(false);
    } catch (error) {
      console.error('Failed to update tachograph status:', error);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteTachograph(tachograph.id).unwrap();
      navigate(paths.fleetTachographs);
    } catch (error) {
      console.error('Failed to delete tachograph:', error);
    }
  };

  const isRevisionExpiring = (date?: string) => {
    if (!date) return false;
    const revisionDate = new Date(date);
    const today = new Date();
    const daysDiff = Math.ceil((revisionDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
    return daysDiff <= 30 && daysDiff >= 0;
  };

  const isRevisionExpired = (date?: string) => {
    if (!date) return false;
    const revisionDate = new Date(date);
    const today = new Date();
    return revisionDate.getTime() < today.getTime();
  };

  return (
    <>
      <Card className="mb-3">
        <Card.Header className="bg-body-tertiary">
          <h5 className="mb-0">
            <FontAwesomeIcon icon="gears" className="me-2 text-primary" />
            Azioni Rapide
          </h5>
        </Card.Header>
        <Card.Body>
          <div className="d-grid gap-2">
            <Button
              variant={tachograph.isActive ? 'warning' : 'success'}
              onClick={() => setShowActivationModal(true)}
            >
              <FontAwesomeIcon
                icon={tachograph.isActive ? 'ban' : 'check-circle'}
                className="me-2"
              />
              {tachograph.isActive ? 'Disattiva' : 'Attiva'} Tachigrafo
            </Button>

            {(isRevisionExpiring(tachograph.scadenzaRevisione) || isRevisionExpired(tachograph.scadenzaRevisione)) && (
              <Button variant="info">
                <FontAwesomeIcon icon="calendar-alt" className="me-2" />
                Programma Revisione
              </Button>
            )}

            <Button variant="primary">
              <FontAwesomeIcon icon="print" className="me-2" />
              Stampa Scheda
            </Button>

            <Button variant="secondary">
              <FontAwesomeIcon icon="file-export" className="me-2" />
              Esporta Dati
            </Button>

            <hr className="my-2" />

            <Button
              variant="outline-danger"
              size="sm"
              onClick={() => setShowDeleteModal(true)}
            >
              <FontAwesomeIcon icon="trash" className="me-2" />
              Elimina Tachigrafo
            </Button>
          </div>
        </Card.Body>
      </Card>

      {/* Activation Modal */}
      <Modal show={showActivationModal} onHide={() => setShowActivationModal(false)} centered>
        <Modal.Header>
          <Modal.Title>
            {tachograph.isActive ? 'Disattiva' : 'Attiva'} Tachigrafo
          </Modal.Title>
          <FalconCloseButton onClick={() => setShowActivationModal(false)} />
        </Modal.Header>
        <Modal.Body>
          <p>
            Sei sicuro di voler {tachograph.isActive ? 'disattivare' : 'attivare'} il tachigrafo{' '}
            <strong>{tachograph.nome}</strong> (Targa: {tachograph.targa})?
          </p>
          {tachograph.isActive && (
            <p className="text-warning mb-0">
              Il tachigrafo non sarà più disponibile fino a quando non verrà riattivato.
            </p>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="secondary"
            onClick={() => setShowActivationModal(false)}
            disabled={isUpdating}
          >
            Annulla
          </Button>
          <Button
            variant={tachograph.isActive ? 'warning' : 'success'}
            onClick={handleToggleActivation}
            disabled={isUpdating}
          >
            {isUpdating ? 'Attendere...' : tachograph.isActive ? 'Disattiva' : 'Attiva'}
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Delete Modal */}
      <Modal show={showDeleteModal} onHide={() => setShowDeleteModal(false)} centered>
        <Modal.Header>
          <Modal.Title>Elimina Tachigrafo</Modal.Title>
          <FalconCloseButton onClick={() => setShowDeleteModal(false)} />
        </Modal.Header>
        <Modal.Body>
          <p>
            Sei sicuro di voler eliminare definitivamente il tachigrafo{' '}
            <strong>{tachograph.nome}</strong> (Targa: {tachograph.targa})?
          </p>
          <p className="text-danger fw-bold mb-0">
            <FontAwesomeIcon icon="exclamation-triangle" className="me-2" />
            Questa azione non può essere annullata!
          </p>
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="secondary"
            onClick={() => setShowDeleteModal(false)}
            disabled={isDeleting}
          >
            Annulla
          </Button>
          <Button
            variant="danger"
            onClick={handleDelete}
            disabled={isDeleting}
          >
            {isDeleting ? (
              <>
                <FontAwesomeIcon icon="spinner" spin className="me-2" />
                Eliminazione...
              </>
            ) : (
              <>
                <FontAwesomeIcon icon="trash" className="me-2" />
                Elimina Definitivamente
              </>
            )}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default TachographActions;