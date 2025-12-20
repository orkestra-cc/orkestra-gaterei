import { Button, Col, Dropdown, Form, Row } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useAdvanceTableContext } from 'providers/AdvanceTableProvider';
import IconButton from 'components/common/IconButton';
import AdvanceTableSearchBox from 'components/common/advance-table/AdvanceTableSearchBox';
import { useState } from 'react';
import { arrayToCSV, downloadCSV, formatDateForCSV, generateTimestampedFilename } from 'utils/csvExport';
import { User } from 'store/api/userApi';
import CreateUserModal from './CreateUserModal';

const UserTableHeader = () => {
  const { getSelectedRowModel, setColumnFilters, getFilteredRowModel } = useAdvanceTableContext();
  const [selectedRole, setSelectedRole] = useState<string>('Tutti');
  const [showCreateModal, setShowCreateModal] = useState(false);

  const roleFilters = [
    'Tutti',
    'CEO',
    'Sviluppatore',
    'Amministratore',
    'Manager',
    'Operatore',
    'Ospite'
  ];

  const handleRoleFilter = (role: string) => {
    setSelectedRole(role);
    if (role === 'Tutti') {
      setColumnFilters([]);
    } else {
      setColumnFilters([{ id: 'role', value: role.toLowerCase() }]);
    }
  };

  const handleExportCSV = () => {
    // Get filtered rows from the table
    const filteredRows = getFilteredRowModel().rows;

    // Map role values to Italian labels
    const roleLabels: Record<string, string> = {
      ceo: 'CEO',
      developer: 'Sviluppatore',
      administrator: 'Amministratore',
      manager: 'Manager',
      operator: 'Operatore',
      guest: 'Ospite'
    };

    // Transform data for CSV export
    const csvData = filteredRows.map((row: any) => {
      const user = row.original as User;
      return {
        'Nome Completo': user.fullName,
        'Email': user.email,
        'Username': user.username,
        'Ruolo': roleLabels[user.role] || user.role,
        'Stato': user.isActive ? 'Attivo' : 'Inattivo',
        'Ultimo Accesso': formatDateForCSV(user.lastLogin),
        'Creato Il': formatDateForCSV(user.createdAt)
      };
    });

    // Define headers
    const headers = [
      'Nome Completo',
      'Email',
      'Username',
      'Ruolo',
      'Stato',
      'Ultimo Accesso',
      'Creato Il'
    ];

    // Generate CSV
    const csv = arrayToCSV(csvData, headers);

    // Download file
    const filename = generateTimestampedFilename('utenti');
    downloadCSV(csv, filename);
  };

  return (
    <div className="d-lg-flex justify-content-between">
      <Row className="flex-between-center gy-2 px-x1">
        <Col xs="auto" className="pe-0">
          <h6 className="mb-0">Gestione utenti</h6>
        </Col>
        <Col xs="auto">
          <AdvanceTableSearchBox
            className="input-search-width"
            placeholder="Cerca per nome/email"
          />
        </Col>
      </Row>
      <div className="border-bottom border-200 my-3"></div>
      <div className="d-flex align-items-center justify-content-between justify-content-lg-end px-x1">
        <Dropdown className="font-sans-serif">
          <Dropdown.Toggle
            variant="falcon-default"
            size="sm"
            className="text-600"
          >
            <FontAwesomeIcon icon="filter" transform="shrink-4" className="me-2" />
            <span className="d-none d-sm-inline-block">{selectedRole}</span>
          </Dropdown.Toggle>
          <Dropdown.Menu className="border py-2">
            {roleFilters.map((role) => (
              <Dropdown.Item
                key={role}
                onClick={() => handleRoleFilter(role)}
                className={selectedRole === role ? 'active' : ''}
              >
                {role}
                {selectedRole === role && (
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
          <div id="users-actions">
            <IconButton
              variant="falcon-default"
              size="sm"
              icon="plus"
              transform="shrink-3"
              iconAlign="middle"
              onClick={() => setShowCreateModal(true)}
            >
              <span className="d-none d-sm-inline-block d-xl-none d-xxl-inline-block ms-1">
                Nuovo utente
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
                  <Dropdown.Item>Esporta</Dropdown.Item>
                  <Dropdown.Item>Importa</Dropdown.Item>
                  <Dropdown.Divider />
                  <Dropdown.Item className="text-danger">Cancella tutto</Dropdown.Item>
                </div>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        )}
      </div>
      <CreateUserModal
        show={showCreateModal}
        onHide={() => setShowCreateModal(false)}
      />
    </div>
  );
};

export default UserTableHeader;