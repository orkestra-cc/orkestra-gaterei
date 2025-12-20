
import WidgetSectionTitle from './WidgetSectionTitle';
import { Col, Row } from 'react-bootstrap';
import Error404 from 'components/errors/Error404';
import Error500 from 'components/errors/Error500';

const ErrorsWidgets = () => {
  return (
    <>
      <WidgetSectionTitle
        icon="exclamation-circle"
        title="Errori"
        subtitle="Visualizza le tue pagine di errore con card eleganti."
        transform="shrink-4"
        className="mb-4 mt-6"
      />

      <Row className="g-3 mb-3">
        <Col lg={6}>
          <Error404 />
        </Col>
        <Col lg={6}>
          <Error500 />
        </Col>
      </Row>
    </>
  );
};

export default ErrorsWidgets;
