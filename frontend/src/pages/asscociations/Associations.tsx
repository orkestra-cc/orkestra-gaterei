import Association from 'pages/asscociations/Association';

import { Card, Col, Row } from 'react-bootstrap';
import associationsData from 'data/associations';

const Associations = ({
  associations = associationsData,
  colBreakpoints = { sm: 6, md: 4 },
  ...rest
}) => {
  return (
    <Card {...rest}>
      <Card.Header className="bg-body-tertiary">
        <h5 className="mb-0">Associations</h5>
      </Card.Header>
      <Card.Body className="fs-10">
        <Row className="g-3">
          {associations.map(association => (
            <Col key={association.title} {...colBreakpoints}>
              <Association
                image={association.image}
                title={association.title}
                description={association.description}
                to={association.to}
              />
            </Col>
          ))}
        </Row>
      </Card.Body>
    </Card>
  );
};

export default Associations;
