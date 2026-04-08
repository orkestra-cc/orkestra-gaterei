import { Col, Form, Row } from 'react-bootstrap';

interface ModuleTableHeaderProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  categoryFilter: string;
  onCategoryChange: (value: string) => void;
  statusFilter: string;
  onStatusChange: (value: string) => void;
}

const ModuleTableHeader: React.FC<ModuleTableHeaderProps> = ({
  searchTerm,
  onSearchChange,
  categoryFilter,
  onCategoryChange,
  statusFilter,
  onStatusChange,
}) => {
  return (
    <Row className="align-items-center g-3">
      <Col xs="auto">
        <h5 className="mb-0">Module Management</h5>
      </Col>
      <Col>
        <Form.Control
          type="search"
          placeholder="Search modules..."
          size="sm"
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </Col>
      <Col xs="auto">
        <Form.Select
          size="sm"
          value={categoryFilter}
          onChange={(e) => onCategoryChange(e.target.value)}
        >
          <option value="">All Categories</option>
          <option value="core">Core</option>
          <option value="toggleable">Toggleable</option>
          <option value="external">External</option>
        </Form.Select>
      </Col>
      <Col xs="auto">
        <Form.Select
          size="sm"
          value={statusFilter}
          onChange={(e) => onStatusChange(e.target.value)}
        >
          <option value="">All Statuses</option>
          <option value="running">Running</option>
          <option value="failed">Failed</option>
          <option value="disabled">Disabled</option>
        </Form.Select>
      </Col>
    </Row>
  );
};

export default ModuleTableHeader;
