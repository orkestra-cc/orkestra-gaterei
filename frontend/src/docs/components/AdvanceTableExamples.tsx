
import { Button } from 'react-bootstrap';
import PageHeader from 'components/common/PageHeader';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import FalconComponentCard from 'components/common/FalconComponentCard';
import IconButton from 'components/common/IconButton';
import AdvanceTable from 'components/common/advance-table/AdvanceTable';
import AdvanceTableFooter from 'components/common/advance-table/AdvanceTableFooter';
import AdvanceTableSearchBox from 'components/common/advance-table/AdvanceTableSearchBox';
import AdvanceTablePagination from 'components/common/advance-table/AdvanceTablePagination';
import useAdvanceTable from 'hooks/ui/useAdvanceTable';
import AdvanceTableProvider, {
  useAdvanceTableContext
} from 'providers/AdvanceTableProvider';

const data = `const columns = [
  {
    accessorKey: 'name',
    header: 'Name',
    meta: {
      headerProps: { className: 'text-900' }
    },
  },
  {
    accessorKey: 'email',
    header: 'Email',
    meta: {
      headerProps: { className: 'text-900' }
    },
  },
  {
    accessorKey: 'age',
    header: 'Age',
    meta: {
      headerProps: { className: 'text-900' }
    },
  }
];

const data = [
  {
    name: 'Anna',
    email: 'anna@example.com',
    age: 18
  },
  {
    name: 'Homer',
    email: 'homer@example.com',
    age: 35
  },
  {
    name: 'Oscar',
    email: 'oscar@example.com',
    age: 52
  },
  {
    name: 'Emily',
    email: 'emily@example.com',
    age: 30
  },
  {
    name: 'Jara',
    email: 'jara@example.com',
    age: 25
  },
  {
    name: 'Clark',
    email: 'clark@example.com',
    age: 39
  },
  {
    name: 'Jennifer',
    email: 'jennifer@example.com',
    age: 52
  },
  {
    name: 'Tony',
    email: 'tony@example.com',
    age: 30
  },
  {
    name: 'Tom',
    email: 'tom@example.com',
    age: 25
  },
  {
    name: 'Michael',
    email: 'michael@example.com',
    age: 39
  },
  {
    name: 'Antony',
    email: 'antony@example.com',
    age: 39
  },
  {
    name: 'Raymond',
    email: 'raymond@example.com',
    age: 52
  },
  {
    name: 'Marie',
    email: 'marie@example.com',
    age: 30
  },
  {
    name: 'Cohen',
    email: 'cohen@example.com',
    age: 25
  },
  {
    name: 'Rowen',
    email: 'rowen@example.com',
    age: 39
  }
];`;

const exampleCode = `${data}

function AdvanceTableExample() {

  const table = useAdvanceTable({
    data,
    columns,
    sortable: true,
    pagination: true,
    perPage: 5
  });

  return(
    <AdvanceTableProvider {...table}>
      <AdvanceTable
        headerClassName="bg-200 text-nowrap align-middle"
        rowClassName="align-middle white-space-nowrap"
        tableProps={{
          bordered: true,
          striped: true,
          className: 'fs-10 mb-0 overflow-hidden'
        }}
      />
      <div className="mt-3">
        <AdvanceTableFooter
          rowInfo
          navButtons
          rowsPerPageSelection
        />
      </div>
    </AdvanceTableProvider>
  )
}

render(<AdvanceTableExample />)
`;

const paginationNumberingCode = `${data}

function AdvanceTableExample() {

  const table = useAdvanceTable({
    data,
    columns,
    sortable: true,
    pagination: true,
    perPage: 5
  });

  return(
    <AdvanceTableProvider {...table}>
      <AdvanceTable
        headerClassName="bg-200 text-nowrap align-middle"
        rowClassName="align-middle white-space-nowrap"
        tableProps={{
          bordered: true,
          striped: true,
          className: 'fs-10 mb-0 overflow-hidden'
        }}
      />
      <div className="mt-3">
        <AdvanceTablePagination />
      </div>
    </AdvanceTableProvider>
  )
}

render(<AdvanceTableExample />)
`;

const selectionCode = `const columns = [
  {
    accessorKey: 'name',
    header: 'Name',
    meta: {
      headerProps: { className: 'text-900' }
    }
  },
  {
    accessorKey: 'email',
    header: 'Email',
    meta: {
      headerProps: { className: 'text-900' },
    },
    cell: ({ row: { original } }) => {
      const { email } = original;;
      return(
        <a href={'mailto:' + email}>
          {email}
        </a>
      )
    }
  },
  {
    accessorKey: 'age',
    header: 'Age',
    meta: {
      headerProps: { className: 'text-900' },
      cellProps:{
        className:'fw-medium'
      }
    },
  }
];

const data = [
  {
    name: 'Anna',
    email: 'anna@example.com',
    age: 18
  },
  {
    name: 'Homer',
    email: 'homer@example.com',
    age: 35
  },
  {
    name: 'Oscar',
    email: 'oscar@example.com',
    age: 52
  },
  {
    name: 'Emily',
    email: 'emily@example.com',
    age: 30
  },
  {
    name: 'Jara',
    email: 'jara@example.com',
    age: 25
  }
];

function BulAction(){
  const { getSelectedRowModel } = useAdvanceTableContext();
  return (
    <Row className="flex-between-center mb-3">
      <Col xs={4} sm="auto" className="d-flex align-items-center pe-0">
        <h5 className="fs-9 mb-0 text-nowrap py-2 py-xl-0">
          {
            getSelectedRowModel().rows.length > 0 ?
            'You have selected ' + getSelectedRowModel().rows.length + ' rows' 
            :
            'Selection Example'
          }
        </h5>
      </Col>
      <Col xs={8} sm="auto" className="ms-auto text-end ps-0">
        {getSelectedRowModel().rows.length > 0 ? (
          <div className="d-flex">
            <Form.Select size="sm" aria-label="Bulk actions">
              <option>Bulk Actions</option>
              <option value="refund">Refund</option>
              <option value="delete">Delete</option>
              <option value="archive">Archive</option>
            </Form.Select>
            <Button
              type="button"
              variant="falcon-default"
              size="sm"
              className="ms-2"
            >
              Apply
            </Button>
          </div>
          ) : (
            <div id="orders-actions">
              <IconButton
                variant="falcon-default"
                size="sm"
                icon="plus"
                transform="shrink-3"
                className='me-2'
              >
                <span className="d-none d-sm-inline-block ms-1">New</span>
              </IconButton>
              <IconButton
                variant="falcon-default"
                size="sm"
                icon="external-link-alt"
                transform="shrink-3"
              >
                <span className="d-none d-sm-inline-block ms-1">Export</span>
              </IconButton>
            </div>
          )}
      </Col>
    </Row>
  );
};

function AdvanceTableExample() {

  const table = useAdvanceTable({
    data,
    columns,
    sortable: true,
    selection: true,
    pagination: true,
    perPage: 5,
    selectionColumnWidth: 30
  });

  return(
    <AdvanceTableProvider {...table}>
      <BulAction />
      <AdvanceTable
        headerClassName="bg-200 text-nowrap align-middle"
        rowClassName="align-middle white-space-nowrap"
        tableProps={{
          striped: true,
          className: 'fs-10 mb-0 overflow-hidden'
        }}
      />
    </AdvanceTableProvider>
  )
}

render(<AdvanceTableExample />)
`;

const customCellCode = `const columns = [
  {
    accessorKey: 'name',
    header: 'Name',
    meta: {
      headerProps: { className: 'text-900' }
    }
  },
  {
    accessorKey: 'email',
    header: 'Email',
    meta: {
      headerProps: { className: 'text-900' },
    },
    cell: ({ row: { original } }) => {
      const { email } = original;
      return(
        <a href={'mailto:' + email}>
          {email}
        </a>
      )
    }
  },
  {
    accessorKey: 'age',
    header: 'Age',
    meta: {
      headerProps: { className: 'text-900' },
    },
    cellProps:{
      className:'fw-medium'
    }
  }
];

const data = [
  {
    name: 'Anna',
    email: 'anna@example.com',
    age: 18
  },
  {
    name: 'Homer',
    email: 'homer@example.com',
    age: 35
  },
  {
    name: 'Oscar',
    email: 'oscar@example.com',
    age: 52
  },
  {
    name: 'Emily',
    email: 'emily@example.com',
    age: 30
  },
  {
    name: 'Jara',
    email: 'jara@example.com',
    age: 25
  }
];

function AdvanceTableExample() {

  const table = useAdvanceTable({
    data,
    columns,
    sortable: true
  });

  return(
    <AdvanceTableProvider {...table}>
      <AdvanceTable
        headerClassName="bg-200 text-nowrap align-middle"
        rowClassName="align-middle white-space-nowrap"
        tableProps={{
          bordered: true,
          striped: true,
          className: 'fs-10 mb-0 overflow-hidden'
        }}
      />
    </AdvanceTableProvider>
  )
}

render(<AdvanceTableExample />)
`;

const searchableTableCode = `${data}

function AdvanceTableExample() {

  const table = useAdvanceTable({
    data,
    columns,
    sortable: true,
    pagination: true,
    perPage: 5
  });

  return (
    <AdvanceTableProvider {...table}>
      <Row className="flex-end-center mb-3">
        <Col xs="auto" sm={6} lg={4}>
          <AdvanceTableSearchBox />
        </Col>
      </Row>
      <AdvanceTable
        headerClassName="bg-200 text-nowrap align-middle"
        rowClassName="align-middle white-space-nowrap"
        tableProps={{
          bordered: true,
          striped: true,
          className: 'fs-10 mb-0 overflow-hidden'
        }}
      />
      <div className="mt-3">
        <AdvanceTableFooter
          rowInfo
          navButtons
          rowsPerPageSelection
        />
      </div>
    </AdvanceTableProvider>
  );
}

render(<AdvanceTableExample />)`;

const AdvanceTableExamples = () => (
  <>
    <PageHeader
      title="Advance Tables"
      description={`React-Falcon uses <strong>React Table</strong> for advance features of table. React Table is a collection of hooks for <strong>building powerful tables and datagrid experiences</strong>. These hooks are lightweight, composable, and ultra-extensible, but <strong>do not render any markup or styles for you</strong>. This effectively means that React Table is a "headless" UI library"`}
      className="mb-3"
    >
      <Button
        href="https://tanstack.com/table/v8/docs/introduction"
        target="_blank"
        variant="link"
        size="sm"
        className="ps-0"
      >
        React Table Documentation
        <FontAwesomeIcon icon="chevron-right" className="ms-1 fs-11" />
      </Button>
    </PageHeader>

    <FalconComponentCard>
      <FalconComponentCard.Header title="Example" light={false} />
      <FalconComponentCard.Body
        code={exampleCode}
        scope={{
          useAdvanceTable,
          AdvanceTableProvider,
          AdvanceTable,
          AdvanceTableFooter
        }}
        language="jsx"
        noInline
        noLight
      />
    </FalconComponentCard>

    <FalconComponentCard>
      <FalconComponentCard.Header
        title="Pagination with numbering"
        light={false}
      />
      <FalconComponentCard.Body
        code={paginationNumberingCode}
        scope={{
          useAdvanceTable,
          AdvanceTableProvider,
          AdvanceTable,
          AdvanceTablePagination
        }}
        language="jsx"
        noInline
        noLight
      />
    </FalconComponentCard>

    <FalconComponentCard>
      <FalconComponentCard.Header
        title="Searchable Table"
        light={false}
        className="border-bottom border-200"
      />
      <FalconComponentCard.Body
        code={searchableTableCode}
        scope={{
          useAdvanceTable,
          AdvanceTableProvider,
          AdvanceTable,
          AdvanceTableFooter,
          AdvanceTableSearchBox
        }}
        language="jsx"
        noInline
        noLight
      />
    </FalconComponentCard>

    <FalconComponentCard>
      <FalconComponentCard.Header
        title="Custom Cell"
        light={false}
        className="border-bottom border-200"
      />
      <FalconComponentCard.Body
        code={customCellCode}
        scope={{
          useAdvanceTable,
          AdvanceTableProvider,
          AdvanceTable,
          AdvanceTableFooter,
          AdvanceTableSearchBox
        }}
        language="jsx"
        noInline
        noLight
      />
    </FalconComponentCard>

    <FalconComponentCard className="mb-0">
      <FalconComponentCard.Header
        title="Selection"
        light={false}
        className="border-bottom border-200"
      />
      <FalconComponentCard.Body
        code={selectionCode}
        scope={{
          useAdvanceTable,
          AdvanceTableProvider,
          AdvanceTable,
          AdvanceTableFooter,
          useAdvanceTableContext,
          IconButton
        }}
        language="jsx"
        noInline
        noLight
      />
    </FalconComponentCard>
  </>
);

export default AdvanceTableExamples;
