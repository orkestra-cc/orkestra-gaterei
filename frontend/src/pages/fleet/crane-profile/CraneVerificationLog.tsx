
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card, Badge, ListGroup, Button } from 'react-bootstrap';
import { FaCheckCircle, FaExclamationTriangle, FaClipboardCheck, FaClock, FaCalendarAlt, FaFileAlt } from 'react-icons/fa';

interface CraneVerificationLogProps {
  craneId: string;
  className?: string;
}

// Mock data for verification activities (in real app, this would come from API)
const mockVerificationData = [
  {
    id: '1',
    type: 'verifica_periodica',
    action: 'Verifica Periodica Annuale Completata',
    timestamp: '2024-01-15T10:30:00',
    description: 'Verifica completa secondo normativa INAIL. Esito positivo.',
    status: 'completed',
    operator: 'Ing. Mario Rossi',
    icon: <FaClipboardCheck />
  },
  {
    id: '2',
    type: 'verifica_trimestrale',
    action: 'Verifica Trimestrale Funi',
    timestamp: '2024-01-10T14:20:00',
    description: 'Controllo integrità funi e catene. Nessuna anomalia rilevata.',
    status: 'completed',
    operator: 'Tech. Luigi Bianchi',
    icon: <FaCheckCircle />
  },
  {
    id: '3',
    type: 'ispezione',
    action: 'Ispezione Straordinaria',
    timestamp: '2023-12-20T09:15:00',
    description: 'Ispezione post-manutenzione del sistema idraulico.',
    status: 'completed',
    operator: 'Ing. Giuseppe Verdi',
    icon: <FaFileAlt />
  },
  {
    id: '4',
    type: 'programmata',
    action: 'Verifica Programmata',
    timestamp: '2024-03-15T00:00:00',
    description: 'Prossima verifica trimestrale programmata',
    status: 'scheduled',
    icon: <FaCalendarAlt />
  },
  {
    id: '5',
    type: 'scadenza',
    action: 'Avviso Scadenza',
    timestamp: '2023-11-30T11:00:00',
    description: 'Verifica in scadenza entro 30 giorni',
    status: 'warning',
    icon: <FaExclamationTriangle />
  }
];

const CraneVerificationLog: React.FC<CraneVerificationLogProps> = ({ craneId, className = '' }) => {
  const formatDate = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffInHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60));

    // Future date
    if (diffInHours < 0) {
      const futureDiff = Math.abs(diffInHours);
      if (futureDiff < 24) {
        return `tra ${futureDiff} ore`;
      } else if (futureDiff < 720) {
        return `tra ${Math.floor(futureDiff / 24)} giorni`;
      } else {
        return date.toLocaleDateString('it-IT', {
          day: 'numeric',
          month: 'short',
          year: 'numeric'
        });
      }
    }

    // Past date
    if (diffInHours < 24) {
      return `${diffInHours} ore fa`;
    } else if (diffInHours < 48) {
      return '1 giorno fa';
    } else if (diffInHours < 720) {
      return `${Math.floor(diffInHours / 24)} giorni fa`;
    } else {
      return date.toLocaleDateString('it-IT', {
        day: 'numeric',
        month: 'short',
        year: 'numeric'
      });
    }
  };

  const getStatusBadge = (type: string) => {
    const badges: Record<string, { bg: string; text: string }> = {
      verifica_periodica: { bg: 'success', text: 'Verifica Annuale' },
      verifica_trimestrale: { bg: 'primary', text: 'Verifica Trimestrale' },
      ispezione: { bg: 'info', text: 'Ispezione' },
      programmata: { bg: 'warning', text: 'Programmata' },
      scadenza: { bg: 'danger', text: 'In Scadenza' }
    };

    const badge = badges[type] || { bg: 'secondary', text: 'Altro' };
    return <Badge bg={badge.bg}>{badge.text}</Badge>;
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      completed: 'success',
      scheduled: 'warning',
      warning: 'danger',
      info: 'info'
    };
    return colors[status] || 'secondary';
  };

  return (
    <Card className={className}>
      <Card.Header className="bg-body-tertiary d-flex justify-content-between align-items-center">
        <h5 className="mb-0">
          <FaClock className="me-2" />
          Cronologia Verifiche
        </h5>
        <Button variant="falcon-primary" size="sm">
          <FontAwesomeIcon icon="plus" className="me-1" />
          Nuova Verifica
        </Button>
      </Card.Header>
      <Card.Body className="p-0">
        <ListGroup variant="flush">
          {mockVerificationData.map((activity, index) => (
            <ListGroup.Item key={activity.id} className="px-3 py-3">
              <div className="d-flex">
                <div className="me-3">
                  <div
                    className={`icon-circle bg-${getStatusColor(activity.status)} text-white d-flex align-items-center justify-content-center`}
                    style={{ width: '32px', height: '32px', borderRadius: '50%', fontSize: '14px' }}
                  >
                    {activity.icon}
                  </div>
                  {index < mockVerificationData.length - 1 && (
                    <div
                      className="bg-300 mx-auto"
                      style={{ width: '2px', height: '40px', marginTop: '8px' }}
                    />
                  )}
                </div>
                <div className="flex-1">
                  <div className="d-flex justify-content-between align-items-start mb-1">
                    <div>
                      <h6 className="mb-0 text-900">{activity.action}</h6>
                      <p className="fs-10 text-600 mb-1">
                        {activity.description}
                      </p>
                      {activity.operator && (
                        <small className="text-muted">
                          <FontAwesomeIcon icon="user" className="me-1" />
                          {activity.operator}
                        </small>
                      )}
                    </div>
                    <div className="text-end">
                      {getStatusBadge(activity.type)}
                      <div className="fs-11 text-muted mt-1">
                        {formatDate(activity.timestamp)}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </ListGroup.Item>
          ))}
        </ListGroup>
      </Card.Body>
      <Card.Footer className="bg-body-tertiary p-3">
        <Button variant="link" size="sm" className="p-0">
          Vedi tutte le verifiche
          <FontAwesomeIcon icon="angle-right" className="ms-1" />
        </Button>
      </Card.Footer>
    </Card>
  );
};

export default CraneVerificationLog;