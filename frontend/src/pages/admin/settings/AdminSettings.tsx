import { Card, Col, Row } from 'react-bootstrap';
import { Link } from 'react-router';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import PageHeader from 'components/common/PageHeader';

interface SettingsCardProps {
  icon: string;
  title: string;
  description: string;
  to: string;
}

const SettingsCard: React.FC<SettingsCardProps> = ({ icon, title, description, to }) => (
  <Col lg={4} md={6}>
    <Card as={Link} to={to} className="h-100 text-decoration-none hover-card">
      <Card.Body className="text-center py-4">
        <FontAwesomeIcon
          icon={icon as any}
          className="text-primary mb-3"
          style={{ fontSize: '2.5rem' }}
        />
        <h5 className="text-900">{title}</h5>
        <p className="text-600 mb-0 fs-10">{description}</p>
      </Card.Body>
    </Card>
  </Col>
);

const AdminSettings = () => {
  return (
    <>
      <PageHeader
        title="Impostazioni"
        description="Configurazione sistema e preferenze"
        className="mb-3"
      />

      {/* Billing Settings */}
      <h6 className="text-uppercase text-600 mb-3">Fatturazione</h6>
      <Row className="g-3 mb-4">
        <SettingsCard
          icon="building"
          title="Aziende Emittenti"
          description="Gestisci le aziende per l'emissione delle fatture elettroniche"
          to="/admin/settings/companies"
        />
      </Row>

      {/* System Settings - Coming Soon */}
      <h6 className="text-uppercase text-600 mb-3">Sistema</h6>
      <Row className="g-3">
        <Col lg={12}>
          <Card>
            <Card.Body className="text-center py-5">
              <FontAwesomeIcon
                icon="cog"
                className="text-400 mb-3"
                style={{ fontSize: '2rem' }}
              />
              <h5 className="text-700">Altre Impostazioni</h5>
              <p className="text-500 mb-0">
                Ulteriori opzioni di configurazione saranno disponibili qui.
              </p>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </>
  );
};

export default AdminSettings;
