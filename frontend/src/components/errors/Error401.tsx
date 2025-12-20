
import { Card, Button } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Link, useNavigate, useLocation } from 'react-router';
import { faHome, faSignOutAlt, faShieldAlt, faArrowLeft } from '@fortawesome/free-solid-svg-icons';
import { useAuth } from 'hooks/auth/useAuthRTK';

interface Error401Props {
  requiredRole?: string;
  userRole?: string;
  message?: string;
}

const Error401: React.FC<Error401Props> = ({
  requiredRole,
  userRole,
  message
}) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user } = useAuth();

  // Get error context from navigation state
  const {
    requiredPermissions,
    accessDeniedReason,
    from
  } = location.state || {};

  const handleGoBack = () => {
    navigate(-1);
  };

  const handleLogout = () => {
    // This will be handled by the logout functionality in your auth system
    window.location.href = '/login';
  };

  const getRoleDisplayName = (role?: string) => {
    const roleMap: Record<string, string> = {
      'developer': 'Sviluppatore',
      'ceo': 'CEO',
      'administrator': 'Amministratore',
      'manager': 'Manager',
      'operator': 'Operatore',
      'guest': 'Ospite'
    };
    return role ? roleMap[role] || role : 'Sconosciuto';
  };

  const getErrorMessage = () => {
    if (message) return message;
    if (accessDeniedReason === 'insufficient_permissions') {
      return 'Permessi Insufficienti';
    }
    return 'Accesso Negato';
  };

  const getDetailedMessage = () => {
    const currentUserRole = userRole || user?.role;

    let details = 'Non hai i permessi necessari per accedere a questa pagina.';

    if (currentUserRole) {
      details += `\n\nIl tuo ruolo attuale: ${getRoleDisplayName(currentUserRole)}`;
    }

    if (requiredPermissions && requiredPermissions.length > 0) {
      const flattenedPermissions = Array.isArray(requiredPermissions[0])
        ? requiredPermissions.flat()
        : requiredPermissions;

      const requiredRoleNames = flattenedPermissions
        .map(getRoleDisplayName)
        .join(', ');

      details += `\n\nRuoli richiesti: ${requiredRoleNames}`;
    }

    if (requiredRole) {
      details += `\n\nRuolo richiesto: ${getRoleDisplayName(requiredRole)}`;
    }

    if (from?.pathname) {
      details += `\n\nPagina richiesta: ${from.pathname}`;
    }

    return details;
  };

  return (
    <Card className="text-center">
      <Card.Body className="p-5">
        <div className="display-1 text-warning fs-error">
          <FontAwesomeIcon icon={faShieldAlt} />
        </div>
        <div className="display-4 text-800 mt-3">401</div>
        <p className="lead mt-4 text-800 font-sans-serif fw-semibold">
          {getErrorMessage()}
        </p>
        <hr />
        <div className="text-600">
          <p style={{ whiteSpace: 'pre-line' }}>
            {getDetailedMessage()}
          </p>
        </div>
        <div className="mt-4">
          <Button
            variant="outline-secondary"
            size="sm"
            className="me-2"
            onClick={handleGoBack}
          >
            <FontAwesomeIcon icon={faArrowLeft} className="me-2" />
            Torna Indietro
          </Button>
          <Link className="btn btn-primary btn-sm me-2" to="/">
            <FontAwesomeIcon icon={faHome} className="me-2" />
            Vai alla Home
          </Link>
          <Button
            variant="outline-danger"
            size="sm"
            onClick={handleLogout}
          >
            <FontAwesomeIcon icon={faSignOutAlt} className="me-2" />
            Logout
          </Button>
        </div>
        <div className="mt-4 pt-3 border-top">
          <small className="text-500">
            Se pensi che questo sia un errore, contatta l'amministratore di sistema.
          </small>
        </div>
      </Card.Body>
    </Card>
  );
};

export default Error401;