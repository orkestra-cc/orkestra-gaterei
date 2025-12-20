import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import coverSrc from 'assets/img/sidereco/sidereco_azienda_e_servizi_8.jpg';
import Flex from 'components/common/Flex';

import { Col, Row, Badge } from 'react-bootstrap';
import ProfileBanner from './VehicleProfileBanner';
import { VehicleResponse } from 'store/api/vehicleApi';
import {
  FaTruck,
  FaTrailer,
  FaMapMarkerAlt,
  FaCalendarCheck
} from 'react-icons/fa';

interface VehicleBannerProps {
  vehicle: VehicleResponse;
}

const VehicleBanner: React.FC<VehicleBannerProps> = ({ vehicle }) => {
  // Helper function to format date
  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/D';
    return new Date(dateString).toLocaleDateString('it-IT', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  };

  // Helper function to calculate days until revision
  const getDaysUntilRevision = (date?: string) => {
    if (!date) return null;
    const revisionDate = new Date(date);
    const today = new Date();
    const diffTime = revisionDate.getTime() - today.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    return diffDays;
  };

  // Type labels in Italian
  const tipoLabels: Record<string, string> = {
    motrice: 'Motrice',
    rimorchio: 'Rimorchio',
    'semi-rimorchio': 'Semi-rimorchio',
    trattore: 'Trattore',
    semovente: 'Semovente'
  };

  const daysUntilRevision = getDaysUntilRevision(vehicle.scadenzaRevisione);

  return (
    <ProfileBanner>
      <ProfileBanner.Header coverSrc={coverSrc}>
        <div className="position-absolute bottom-0 start-0 p-3 d-flex align-items-center">
          <div
            className="avatar-5xl rounded-circle bg-white d-flex align-items-center justify-content-center shadow-sm"
            style={{ width: '120px', height: '120px' }}
          >
            {vehicle.tipo === 'motrice' || vehicle.tipo === 'trattore' || vehicle.tipo === 'semovente' ? (
              <FaTruck className="text-primary" size={60} />
            ) : (
              <FaTrailer className="text-primary" size={60} />
            )}
          </div>
        </div>
      </ProfileBanner.Header>
      <ProfileBanner.Body>
        <Row className="mt-4">
          <Col lg={8}>
            <Flex alignItems="center" className="mb-2">
              <h4 className="mb-0 me-2">{vehicle.nome}</h4>
              <Badge
                bg={vehicle.isActive ? 'success' : 'secondary'}
                className="ms-2"
              >
                {vehicle.isActive ? 'Attivo' : 'Inattivo'}
              </Badge>
            </Flex>

            <div className="fs-10 fw-medium text-500 mb-2">
              <FontAwesomeIcon icon="id-card" className="me-2" />
              Targa: <span className="text-900 fw-bold">{vehicle.targa}</span>
            </div>

            <p className="text-1000 mb-0">
              <Badge bg="soft-primary" className="me-2">
                {tipoLabels[vehicle.tipo] || vehicle.tipo}
              </Badge>
              {vehicle.luogo && (
                <>
                  <FaMapMarkerAlt className="me-1 text-muted" />
                  <span className="text-muted">Posizione: {vehicle.luogo}</span>
                </>
              )}
            </p>
          </Col>
          <Col lg={4} className="text-lg-end">
            <div className="border-start-lg ps-lg-4">
              <div className="mb-3">
                <h6 className="text-uppercase text-600 mb-0">
                  <FontAwesomeIcon icon="calendar-alt" className="me-2" />
                  Scadenza Revisione
                </h6>
                <div className="fs-5 fw-medium text-1000">
                  {formatDate(vehicle.scadenzaRevisione)}
                </div>
                {daysUntilRevision !== null && (
                  <Badge
                    bg={
                      daysUntilRevision <= 30
                        ? 'warning'
                        : daysUntilRevision < 0
                          ? 'danger'
                          : 'success'
                    }
                    className="mt-1"
                  >
                    {daysUntilRevision < 0
                      ? `Scaduta da ${Math.abs(daysUntilRevision)} giorni`
                      : daysUntilRevision === 0
                        ? 'Scade oggi'
                        : `${daysUntilRevision} giorni rimanenti`}
                  </Badge>
                )}
              </div>

              {vehicle.revisioneProgrammata && (
                <div>
                  <h6 className="text-uppercase text-600 mb-0">
                    <FaCalendarCheck className="me-2" />
                    Revisione Programmata
                  </h6>
                  <div className="fs-5 fw-medium text-1000">
                    {formatDate(vehicle.revisioneProgrammata)}
                  </div>
                </div>
              )}
            </div>
          </Col>
        </Row>
      </ProfileBanner.Body>
    </ProfileBanner>
  );
};

export default VehicleBanner;
