
import { Button, Col, Form, Row } from 'react-bootstrap';
import { useAdvanceTableContext } from 'providers/AdvanceTableProvider';
import IconButton from 'components/common/IconButton';
import AdvanceTableSearchBox from 'components/common/advance-table/AdvanceTableSearchBox';

interface DeadlineReportsHeaderProps {
  onFilterChange: (filterType: string, value: string) => void;
  entityTypeFilter: string;
  statusFilter: string;
}

const DeadlineReportsHeader: React.FC<DeadlineReportsHeaderProps> = ({
  onFilterChange,
  entityTypeFilter,
  statusFilter,
}) => {
  const { getSelectedRowModel } = useAdvanceTableContext();

  return (
    <div className="d-lg-flex justify-content-between">
      <Row className="flex-between-center gy-2 px-x1">
        <Col xs="auto" className="pe-0">
          <h6 className="mb-0">Report Scadenze</h6>
        </Col>
        <Col xs="auto">
          <AdvanceTableSearchBox
            className="input-search-width"
            placeholder="Cerca per nome"
          />
        </Col>
      </Row>
      <div className="border-bottom border-200 my-3"></div>
      <div className="d-flex align-items-center justify-content-between justify-content-lg-end px-x1 flex-wrap gap-2">
        {/* Entity Type Filter */}
        <Form.Select
          size="sm"
          aria-label="Filtra per tipo"
          value={entityTypeFilter}
          onChange={(e) => onFilterChange('entityType', e.target.value)}
          style={{ width: 'auto', minWidth: '150px' }}
        >
          <option value="">Tutti i tipi</option>
          <option value="vehicle">Veicoli</option>
          <option value="user">Utenti</option>
          <option value="medical">Visite Mediche</option>
        </Form.Select>

        {/* Status Filter */}
        <Form.Select
          size="sm"
          aria-label="Filtra per stato"
          value={statusFilter}
          onChange={(e) => onFilterChange('status', e.target.value)}
          style={{ width: 'auto', minWidth: '150px' }}
        >
          <option value="">Tutti gli stati</option>
          <option value="expired">Scaduti</option>
          <option value="warning">In Scadenza</option>
          <option value="ok">OK</option>
        </Form.Select>

        <div
          className="bg-300 mx-2 d-none d-lg-block"
          style={{ width: '1px', height: '29px' }}
        ></div>

        {getSelectedRowModel().rows.length > 0 ? (
          <div className="d-flex">
            <Form.Select size="sm" aria-label="Azioni multiple">
              <option>Azioni multiple</option>
              <option value="export">Esporta selezionati</option>
            </Form.Select>
            <Button
              type="button"
              variant="falcon-default"
              size="sm"
              className="ms-2"
            >
              Applica
            </Button>
          </div>
        ) : (
          <div id="deadline-actions">
            <IconButton
              variant="falcon-default"
              size="sm"
              icon="external-link-alt"
              transform="shrink-3"
              iconAlign="middle"
            >
              <span className="d-none d-sm-inline-block ms-1">
                Esporta
              </span>
            </IconButton>
          </div>
        )}
      </div>
    </div>
  );
};

export default DeadlineReportsHeader;
