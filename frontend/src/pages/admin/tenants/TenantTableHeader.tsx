import { Button, Col, Form, Row } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

interface Props {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  planFilter: string;
  onPlanChange: (value: string) => void;
  includeDeleted: boolean;
  onIncludeDeletedChange: (value: boolean) => void;
  onCreateClick: () => void;
}

const TenantTableHeader: React.FC<Props> = ({
  searchTerm,
  onSearchChange,
  planFilter,
  onPlanChange,
  onCreateClick,
}) => {
  return (
    <Row className="align-items-center g-3">
      <Col xs="auto">
        <h5 className="mb-0">Tenant Management</h5>
      </Col>
      <Col>
        <Form.Control
          type="search"
          size="sm"
          placeholder="Search by name or slug..."
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </Col>
      <Col xs="auto">
        <Form.Select
          size="sm"
          value={planFilter}
          onChange={(e) => onPlanChange(e.target.value)}
        >
          <option value="">All Plans</option>
          <option value="free">Free</option>
          <option value="pro">Pro</option>
          <option value="enterprise">Enterprise</option>
        </Form.Select>
      </Col>
      <Col xs="auto">
        <Button variant="primary" size="sm" onClick={onCreateClick}>
          <FontAwesomeIcon icon="plus" className="me-1" />
          New Tenant
        </Button>
      </Col>
    </Row>
  );
};

export default TenantTableHeader;
