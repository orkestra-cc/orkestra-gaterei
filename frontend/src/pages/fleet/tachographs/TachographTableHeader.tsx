import { Button, Col, Dropdown, Form, Row } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useAdvanceTableContext } from 'providers/AdvanceTableProvider';
import IconButton from 'components/common/IconButton';
import AdvanceTableSearchBox from 'components/common/advance-table/AdvanceTableSearchBox';
import { useState } from 'react';
import { arrayToCSV, downloadCSV, formatDateForCSV, generateTimestampedFilename } from 'utils/csvExport';
import type { TachographResponse } from 'store/api/tachographApi';
import AddTachographModal from './AddTachographModal';

const TachographTableHeader = () => {
  const { getSelectedRowModel, setColumnFilters, getFilteredRowModel } = useAdvanceTableContext();
  const [selectedStatus, setSelectedStatus] = useState<string>('Tutti');
  const [selectedRevision, setSelectedRevision] = useState<string>('Tutti');
  const [showAddModal, setShowAddModal] = useState(false);

  const statusFilters = [
    'Tutti',
    'Attivi',
    'Inattivi'
  ];

  const revisionFilters = [
    { label: 'Tutti', value: 'Tutti' },
    { label: 'Prossimi 7 giorni', value: '7' },
    { label: 'Prossimi 30 giorni', value: '30' },
    { label: 'Prossimi 60 giorni', value: '60' },
    { label: 'Scaduti', value: 'expired' }
  ];

  const handleStatusFilter = (status: string) => {
    setSelectedStatus(status);
    applyFilters(status, selectedRevision);
  };

  const handleRevisionFilter = (revision: string) => {
    setSelectedRevision(revision);
    applyFilters(selectedStatus, revision);
  };

  const applyFilters = (status: string, revision: string) => {
    const filters = [];

    if (status !== 'Tutti') {
      filters.push({ id: 'isActive', value: status === 'Attivi' });
    }

    // Note: Revision filtering would need to be handled differently
    // as it's a server-side filter. This is just for UI display
    if (revision !== 'Tutti') {
      // You might need to implement custom filtering logic here
      // or handle it through the API query parameters
    }

    setColumnFilters(filters);
  };

  const handleExportCSV = () => {
    // Get filtered rows from the table
    const filteredRows = getFilteredRowModel().rows;

    // Transform data for CSV export
    const csvData = filteredRows.map((row: any) => {
      const tachograph = row.original as TachographResponse;
      return {
        'Nome': tachograph.nome,
        'Targa': tachograph.targa,
        'Posizione': tachograph.luogo || '',
        'Stato': tachograph.isActive ? 'Attivo' : 'Inattivo',
        'Scadenza Revisione': formatDateForCSV(tachograph.scadenzaRevisione),
        'Revisione Programmata': formatDateForCSV(tachograph.revisioneProgrammata),
        'Note': tachograph.note || '',
        'Creato Il': formatDateForCSV(tachograph.createdAt),
        'Aggiornato Il': formatDateForCSV(tachograph.updatedAt)
      };
    });

    // Define headers
    const headers = [
      'Nome',
      'Targa',
      'Posizione',
      'Stato',
      'Scadenza Revisione',
      'Revisione Programmata',
      'Note',
      'Creato Il',
      'Aggiornato Il'
    ];

    // Generate CSV
    const csv = arrayToCSV(csvData, headers);

    // Download file
    const filename = generateTimestampedFilename('tachografi');
    downloadCSV(csv, filename);
  };

  return (
    <>
      <div className="d-lg-flex justify-content-between">
      <Row className="flex-between-center gy-2 px-x1">
        <Col xs="auto" className="pe-0">
          <h6 className="mb-0">Gestione Tachografi</h6>
        </Col>
        <Col xs="auto">
          <AdvanceTableSearchBox
            className="input-search-width"
            placeholder="Cerca per nome/targa"
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
            <FontAwesomeIcon icon="calendar-check" transform="shrink-4" className="me-2" />
            <span className="d-none d-sm-inline-block">Revisione: {revisionFilters.find(f => f.value === selectedRevision)?.label || 'Tutti'}</span>
          </Dropdown.Toggle>
          <Dropdown.Menu className="border py-2">
            {revisionFilters.map((filter) => (
              <Dropdown.Item
                key={filter.value}
                onClick={() => handleRevisionFilter(filter.value)}
                className={selectedRevision === filter.value ? 'active' : ''}
              >
                {filter.label}
                {selectedRevision === filter.value && (
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
              <option value="schedule-revision">Programma Revisione</option>
              <option value="export-selected">Esporta Selezionati</option>
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
          <div id="tachographs-actions">
            <IconButton
              variant="falcon-default"
              size="sm"
              icon="plus"
              transform="shrink-3"
              iconAlign="middle"
              onClick={() => setShowAddModal(true)}
            >
              <span className="d-none d-sm-inline-block d-xl-none d-xxl-inline-block ms-1">
                Nuovo Tachigrafo
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
                  <Dropdown.Item>Revisioni in Scadenza</Dropdown.Item>
                  <Dropdown.Item>Revisioni Scadute</Dropdown.Item>
                  <Dropdown.Item>Report Conformità</Dropdown.Item>
                  <Dropdown.Divider />
                  <Dropdown.Item className="text-danger">Cancella tutto</Dropdown.Item>
                </div>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        )}
      </div>
    </div>
      <AddTachographModal
        show={showAddModal}
        onHide={() => setShowAddModal(false)}
      />
    </>
  );
};

export default TachographTableHeader;