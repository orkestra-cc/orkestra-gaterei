import { Button, Col, Dropdown, Form, Row } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useAdvanceTableContext } from 'providers/AdvanceTableProvider';
import IconButton from 'components/common/IconButton';
import AdvanceTableSearchBox from 'components/common/advance-table/AdvanceTableSearchBox';
import { useState } from 'react';
import { arrayToCSV, downloadCSV, formatDateForCSV, generateTimestampedFilename } from 'utils/csvExport';
import type { CraneResponse } from 'store/api/craneApi';
import AddCraneModal from './AddCraneModal';

const CraneTableHeader = () => {
  const { getSelectedRowModel, setColumnFilters, getFilteredRowModel } = useAdvanceTableContext();
  const [selectedType, setSelectedType] = useState<string>('Tutti');
  const [selectedStatus, setSelectedStatus] = useState<string>('Tutti');
  const [selectedVerification, setSelectedVerification] = useState<string>('Tutti');
  const [showAddModal, setShowAddModal] = useState(false);

  const typeFilters = [
    'Tutti',
    'Autogrù',
    'Gru a torre',
    'Gru mobile',
    'Gru fissa'
  ];

  const statusFilters = [
    'Tutti',
    'Attive',
    'Inattive'
  ];

  const verificationFilters = [
    'Tutti',
    'In regola',
    'In scadenza',
    'Scadute'
  ];

  const handleTypeFilter = (type: string) => {
    setSelectedType(type);
    applyFilters(type, selectedStatus, selectedVerification);
  };

  const handleStatusFilter = (status: string) => {
    setSelectedStatus(status);
    applyFilters(selectedType, status, selectedVerification);
  };

  const handleVerificationFilter = (verification: string) => {
    setSelectedVerification(verification);
    applyFilters(selectedType, selectedStatus, verification);
  };

  const applyFilters = (type: string, status: string, verification: string) => {
    const filters = [];

    if (type !== 'Tutti') {
      filters.push({ id: 'tipo', value: type });
    }

    if (status !== 'Tutti') {
      filters.push({ id: 'isActive', value: status === 'Attive' });
    }

    if (verification !== 'Tutti') {
      // This would require custom filtering logic
      if (verification === 'In scadenza') {
        // Filter cranes with verification expiring in 30 days
        filters.push({ id: 'verificationStatus', value: 'expiring' });
      } else if (verification === 'Scadute') {
        filters.push({ id: 'verificationStatus', value: 'expired' });
      } else if (verification === 'In regola') {
        filters.push({ id: 'verificationStatus', value: 'valid' });
      }
    }

    setColumnFilters(filters);
  };

  const handleExportCSV = () => {
    // Get filtered rows from the table
    const filteredRows = getFilteredRowModel().rows;

    // Transform data for CSV export
    const csvData = filteredRows.map((row: any) => {
      const crane = row.original as CraneResponse;
      return {
        'Nome': crane.nome,
        'Tipo': crane.tipo,
        'Matricola': crane.matricola,
        'Mezzo Associato': crane.verificareSuMezzo || '',
        'Stato': crane.isActive ? 'Attiva' : 'Inattiva',
        'Scadenza Verifica': formatDateForCSV(crane.scadenzaVerifica),
        'Note': crane.note || '',
        'Creato Il': formatDateForCSV(crane.createdAt),
        'Aggiornato Il': formatDateForCSV(crane.updatedAt)
      };
    });

    // Define headers
    const headers = [
      'Nome',
      'Tipo',
      'Matricola',
      'Mezzo Associato',
      'Stato',
      'Scadenza Verifica',
      'Note',
      'Creato Il',
      'Aggiornato Il'
    ];

    // Generate CSV
    const csv = arrayToCSV(csvData, headers);

    // Download file
    const filename = generateTimestampedFilename('gru');
    downloadCSV(csv, filename);
  };

  return (
    <>
      <div className="d-lg-flex justify-content-between">
      <Row className="flex-between-center gy-2 px-x1">
        <Col xs="auto" className="pe-0">
          <h6 className="mb-0">Gestione Gru</h6>
        </Col>
        <Col xs="auto">
          <AdvanceTableSearchBox
            className="input-search-width"
            placeholder="Cerca per nome/matricola/tipo"
          />
        </Col>
      </Row>
      <div className="border-bottom border-200 my-3"></div>
      <div className="d-flex align-items-center justify-content-between justify-content-lg-end px-x1">
        <Dropdown className="font-sans-serif me-2">
          <Dropdown.Toggle
            variant="falcon-default"
            size="sm"
            className="text-600"
          >
            <FontAwesomeIcon icon="filter" transform="shrink-4" className="me-2" />
            <span className="d-none d-sm-inline-block">Tipo: {selectedType}</span>
          </Dropdown.Toggle>
          <Dropdown.Menu className="border py-2">
            {typeFilters.map((type) => (
              <Dropdown.Item
                key={type}
                onClick={() => handleTypeFilter(type)}
                className={selectedType === type ? 'active' : ''}
              >
                {type}
                {selectedType === type && (
                  <FontAwesomeIcon
                    icon="check"
                    transform="down-4 shrink-4"
                    className="ms-2"
                  />
                )}
              </Dropdown.Item>
            ))}
          </Dropdown.Menu>
        </Dropdown>
        <Dropdown className="font-sans-serif me-2">
          <Dropdown.Toggle
            variant="falcon-default"
            size="sm"
            className="text-600"
          >
            <FontAwesomeIcon icon="filter" transform="shrink-4" className="me-2" />
            <span className="d-none d-sm-inline-block">Stato: {selectedStatus}</span>
          </Dropdown.Toggle>
          <Dropdown.Menu className="border py-2">
            {statusFilters.map((status) => (
              <Dropdown.Item
                key={status}
                onClick={() => handleStatusFilter(status)}
                className={selectedStatus === status ? 'active' : ''}
              >
                {status}
                {selectedStatus === status && (
                  <FontAwesomeIcon
                    icon="check"
                    transform="down-4 shrink-4"
                    className="ms-2"
                  />
                )}
              </Dropdown.Item>
            ))}
          </Dropdown.Menu>
        </Dropdown>
        <Dropdown className="font-sans-serif">
          <Dropdown.Toggle
            variant="falcon-default"
            size="sm"
            className="text-600"
          >
            <FontAwesomeIcon icon="filter" transform="shrink-4" className="me-2" />
            <span className="d-none d-sm-inline-block">Verifica: {selectedVerification}</span>
          </Dropdown.Toggle>
          <Dropdown.Menu className="border py-2">
            {verificationFilters.map((verification) => (
              <Dropdown.Item
                key={verification}
                onClick={() => handleVerificationFilter(verification)}
                className={selectedVerification === verification ? 'active' : ''}
              >
                {verification}
                {selectedVerification === verification && (
                  <FontAwesomeIcon
                    icon="check"
                    transform="down-4 shrink-4"
                    className="ms-2"
                  />
                )}
              </Dropdown.Item>
            ))}
          </Dropdown.Menu>
        </Dropdown>
        <div
          className="bg-300 mx-3 d-none d-lg-block"
          style={{ width: '1px', height: '29px' }}
        ></div>
        {getSelectedRowModel().rows.length > 0 ? (
          <div className="d-flex">
            <Form.Select size="sm" aria-label="Azioni di gruppo">
              <option>Azioni di gruppo</option>
              <option value="activate">Attiva</option>
              <option value="deactivate">Disattiva</option>
              <option value="delete">Elimina</option>
              <option value="schedule-verification">Programma Verifica</option>
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
          <div id="cranes-actions">
            <IconButton
              variant="falcon-default"
              size="sm"
              icon="plus"
              transform="shrink-3"
              iconAlign="middle"
              onClick={() => setShowAddModal(true)}
            >
              <span className="d-none d-sm-inline-block d-xl-none d-xxl-inline-block ms-1">
                Nuova Gru
              </span>
            </IconButton>
            <IconButton
              variant="falcon-default"
              size="sm"
              icon="external-link-alt"
              transform="shrink-3"
              className="mx-2"
              iconAlign="middle"
              onClick={handleExportCSV}
            >
              <span className="d-none d-sm-inline-block d-xl-none d-xxl-inline-block ms-1">
                Esporta
              </span>
            </IconButton>
            <Dropdown align="end" className="btn-reveal-trigger d-inline-block">
              <Dropdown.Toggle variant="falcon-default" size="sm">
                <FontAwesomeIcon icon="ellipsis-h" className="fs-11" />
              </Dropdown.Toggle>

              <Dropdown.Menu className="border py-0">
                <div className="py-2">
                  <Dropdown.Item as="button" type="button">
                    Visualizza tutto
                  </Dropdown.Item>
                  <Dropdown.Item onClick={handleExportCSV}>Esporta</Dropdown.Item>
                  <Dropdown.Item>Importa</Dropdown.Item>
                  <Dropdown.Divider />
                  <Dropdown.Item>Verifiche in Scadenza</Dropdown.Item>
                  <Dropdown.Item>Report Utilizzo</Dropdown.Item>
                  <Dropdown.Item>Gru per Mezzo</Dropdown.Item>
                  <Dropdown.Divider />
                  <Dropdown.Item className="text-danger">Cancella tutto</Dropdown.Item>
                </div>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        )}
      </div>
    </div>
      <AddCraneModal
        show={showAddModal}
        onHide={() => setShowAddModal(false)}
      />
    </>
  );
};

export default CraneTableHeader;