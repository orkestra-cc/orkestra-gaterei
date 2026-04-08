import { Col, Row } from 'react-bootstrap';
import ModuleTable from './ModuleTable';

const ModuleManagementPage: React.FC = () => {
  return (
    <Row className="g-3">
      <Col xxl={12}>
        <ModuleTable />
      </Col>
    </Row>
  );
};

export default ModuleManagementPage;
