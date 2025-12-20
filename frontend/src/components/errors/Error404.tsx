
import { Card } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Link } from 'react-router';
import { faHome } from '@fortawesome/free-solid-svg-icons';

const Error404 = () => {
  return (
    <Card className="text-center">
      <Card.Body className="p-5">
        <div className="display-1 text-300 fs-error">404</div>
        <p className="lead mt-4 text-800 font-sans-serif fw-semibold">
          La pagina che stai cercando non è stata trovata.
        </p>
        <hr />
        <p>
          Assicurati che l'indirizzo sia corretto e che la pagina non sia stata spostata. Se
          pensi che questo sia un errore,
          <a href="mailto:info@exmaple.com" className="ms-1">
            contattaci
          </a>
          .
        </p>
        <Link className="btn btn-primary btn-sm mt-3" to="/">
          <FontAwesomeIcon icon={faHome} className="me-2" />
          Torna alla home
        </Link>
      </Card.Body>
    </Card>
  );
};

export default Error404;
