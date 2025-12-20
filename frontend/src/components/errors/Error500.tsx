
import { Card } from 'react-bootstrap';

const Error500 = () => (
  <Card className="text-center h-100">
    <Card.Body className="p-5">
      <div className="display-1 text-300 fs-error">500</div>
      <p className="lead mt-4 text-800 font-sans-serif fw-semibold">
        Ops, qualcosa è andato storto!
      </p>
      <hr />
      <p>
        Prova ad aggiornare la pagina, o torna indietro e riprova l'azione.
        Se il problema persiste,
        <a href="mailto:info@exmaple.com" className="ms-1">
          contattaci
        </a>
        .
      </p>
    </Card.Body>
  </Card>
);

export default Error500;
