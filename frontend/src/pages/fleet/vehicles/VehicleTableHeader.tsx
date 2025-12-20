import { Button, Col, Dropdown, Form, Row } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useAdvanceTableContext } from 'providers/AdvanceTableProvider';
import IconButton from 'components/common/IconButton';
import AdvanceTableSearchBox from 'components/common/advance-table/AdvanceTableSearchBox';
import { useState } from 'react';
import { arrayToCSV, downloadCSV, formatDateForCSV, generateTimestampedFilename } from 'utils/csvExport';
import type { VehicleResponse } from 'store/api/vehicleApi';
import AddVehicleModal from './AddVehicleModal';

const VehicleTableHeader = () => {
  const { getSelectedRowModel, setColumnFilters, getFilteredRowModel } = useAdvanceTableContext();
  const [selectedType, setSelectedType] = useState<string>('Tutti');
  const [selectedStatus, setSelectedStatus] = useState<string>('Tutti');
  const [showAddModal, setShowAddModal] = useState(false);

  const typeFilters = [
    'Tutti',
    'Motrice',
    'Rimorchio',
    'Semi-rimorchio',
    'Trattore',
    'Semovente'
  ];

  const statusFilters = [
    'Tutti',
    'Attivi',
    'Inattivi'
  ];

  const handleTypeFilter = (type: string) => {
    setSelectedType(type);
    applyFilters(type, selectedStatus);
  };

  const handleStatusFilter = (status: string) => {
    setSelectedStatus(status);
    applyFilters(selectedType, status);
  };

  const applyFilters = (type: string, status: string) => {
    const filters = [];

    if (type !== 'Tutti') {
      filters.push({ id: 'tipo', value: type.toLowerCase() });
    }

    if (status !== 'Tutti') {
      filters.push({ id: 'isActive', value: status === 'Attivi' });
    }

    setColumnFilters(filters);
  };

  const handleExportCSV = () => {
    // Get filtered rows from the table
    const filteredRows = getFilteredRowModel().rows;

    // Map tipo values to Italian labels
    const tipoLabels: Record<string, string> = {
      motrice: 'Motrice',
      rimorchio: 'Rimorchio',
      'semi-rimorchio': 'Semi-rimorchio',
      trattore: 'Trattore',
      semovente: 'Semovente'
    };

    // Transform data for CSV export
    const csvData = filteredRows.map((row: any) => {
      const vehicle = row.original as VehicleResponse;
      return {
        'Nome': vehicle.nome,
        'Targa': vehicle.targa,
        'Tipo': tipoLabels[vehicle.tipo] || vehicle.tipo,
        'Posizione': vehicle.luogo || '',
        'Stato': vehicle.isActive ? 'Attivo' : 'Inattivo',
        'Scadenza Revisione': formatDateForCSV(vehicle.scadenzaRevisione),
        'Revisione Programmata': formatDateForCSV(vehicle.revisioneProgrammata),
        'Note': vehicle.note || '',
        'Creato Il': formatDateForCSV(vehicle.createdAt),
        'Aggiornato Il': formatDateForCSV(vehicle.updatedAt)
      };
    });

    // Define headers
    const headers = [
      'Nome',
      'Targa',
      'Tipo',
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
    const filename = generateTimestampedFilename('mezzi');
    downloadCSV(csv, filename);
  };

  return (
    <>
      <div className="d-lg-flex justify-content-between">
      <Row className="flex-between-center gy-2 px-x1">
        <Col xs="auto" className="pe-0">
          <h6 className="mb-0">Gestione Mezzi</h6>
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
        <Dropdown className="font-sans-serif">
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
          <div id="vehicles-actions">
            <IconButton
              variant="falcon-default"
              size="sm"
              icon="plus"
              transform="shrink-3"
              iconAlign="middle"
              onClick={() => setShowAddModal(true)}
            >
              <span className="d-none d-sm-inline-block d-xl-none d-xxl-inline-block ms-1">
                Nuovo Mezzo
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
                  <Dropdown.Item>Report Utilizzo</Dropdown.Item>
                  <Dropdown.Divider />
                  <Dropdown.Item className="text-danger">Cancella tutto</Dropdown.Item>
                </div>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        )}
      </div>
    </div>
      <AddVehicleModal
        show={showAddModal}
        onHide={() => setShowAddModal(false)}
      />
    </>
  );
};

export default VehicleTableHeader;