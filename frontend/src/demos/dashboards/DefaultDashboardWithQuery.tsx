
import { Row, Col, Spinner, Alert } from 'react-bootstrap';
import WeeklySales from 'components/dashboards/default/WeeklySales';
import TotalOrder from 'components/dashboards/default/TotalOrder';
import MarketShare from 'components/dashboards/default/MarketShare';
import TotalSales from 'components/dashboards/default/TotalSales';
import RunningProjects from 'components/dashboards/default/RunningProjects';
import StorageStatus from 'components/dashboards/default/StorageStatus';
import SpaceWarning from 'components/dashboards/default/SpaceWarning';
import BestSellingProducts from 'components/dashboards/default/BestSellingProducts';
import SharedFiles from 'components/dashboards/default/SharedFiles';
import ActiveUsers from 'components/dashboards/default/ActiveUsers';
import BandwidthSaved from 'components/dashboards/default/BandwidthSaved';
import TopProducts from 'components/dashboards/default/TopProducts';
import Weather from 'components/dashboards/default/Weather';

// Import TanStack Query hooks
import {
  useDefaultDashboard,
  useWeatherData,
  useBestSellingProducts,
  useStorageStatus
} from 'hooks/useDashboard';

/**
 * Default Dashboard with TanStack Query Integration
 * This is an example of how to migrate dashboard components to use TanStack Query
 */
const DefaultDashboardWithQuery: React.FC = () => {
  // Fetch multiple dashboard data sources
  const dashboardQueries = useDefaultDashboard('30d');
  
  // Extract individual query results
  const [totalSalesQuery, totalOrdersQuery, activeUsersQuery, weeklySalesQuery, bestProductsQuery, storageQuery, weatherQuery] = dashboardQueries;

  // Additional specific queries for other components
  const bestSellingProducts = useBestSellingProducts(8);
  
  // Show loading state if any critical queries are loading
  const isLoading = totalSalesQuery.isLoading || totalOrdersQuery.isLoading;
  
  // Show error if any critical queries failed
  const hasError = totalSalesQuery.error || totalOrdersQuery.error;

  if (isLoading) {
    return (
      <div className="d-flex justify-content-center align-items-center" style={{ minHeight: '400px' }}>
        <div className="text-center">
          <Spinner animation="border" className="mb-3" />
          <p className="text-muted">Loading dashboard data...</p>
        </div>
      </div>
    );
  }

  if (hasError) {
    return (
      <Alert variant="danger">
        <Alert.Heading>Error Loading Dashboard</Alert.Heading>
        <p>Failed to load dashboard data. Please try refreshing the page.</p>
        <div className="d-flex gap-2">
          <button 
            className="btn btn-outline-danger btn-sm"
            onClick={() => {
              totalSalesQuery.refetch();
              totalOrdersQuery.refetch();
            }}
          >
            Retry
          </button>
        </div>
      </Alert>
    );
  }

  return (
    <>
      <Row className="g-3 mb-3">
        <Col md={6} xxl={3}>
          <WeeklySales 
            data={weeklySalesQuery.data} 
            isLoading={weeklySalesQuery.isLoading}
            error={weeklySalesQuery.error}
          />
        </Col>
        <Col md={6} xxl={3}>
          <TotalOrder 
            data={totalOrdersQuery.data}
            isLoading={totalOrdersQuery.isLoading}
            error={totalOrdersQuery.error}
          />
        </Col>
        <Col md={6} xxl={3}>
          <MarketShare 
            data={marketShare} // This could be migrated to query later
            radius={['100%', '87%']} 
          />
        </Col>
        <Col md={6} xxl={3}>
          <Weather 
            data={weatherQuery.data}
            isLoading={weatherQuery.isLoading}
            error={weatherQuery.error}
          />
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        <Col lg={6}>
          <RunningProjects 
            data={runningProjects} // This could be migrated to query later
          />
        </Col>
        <Col lg={6}>
          <TotalSales 
            data={totalSalesQuery.data}
            isLoading={totalSalesQuery.isLoading}
            error={totalSalesQuery.error}
          />
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        <Col lg={6} xl={7} xxl={8}>
          <StorageStatus 
            className="h-lg-100" 
            data={storageQuery.data}
            isLoading={storageQuery.isLoading}
            error={storageQuery.error}
          />
        </Col>
        <Col lg={6} xl={5} xxl={4}>
          <SpaceWarning />
        </Col>
      </Row>

      <Row className="g-3 mb-3">
        <Col lg={7} xl={8}>
          <BestSellingProducts 
            products={bestSellingProducts.data}
            isLoading={bestSellingProducts.isLoading}
            error={bestSellingProducts.error}
          />
        </Col>
        <Col lg={5} xl={4}>
          <SharedFiles files={files} className="h-lg-100" />
        </Col>
      </Row>

      <Row className="g-3">
        <Col sm={6} xxl={3}>
          <ActiveUsers 
            className="h-100" 
            users={activeUsersQuery.data}
            isLoading={activeUsersQuery.isLoading}
            error={activeUsersQuery.error}
          />
        </Col>
        <Col sm={6} xxl={3} className="order-xxl-1">
          <BandwidthSaved />
        </Col>
        <Col xxl={6}>
          <TopProducts 
            data={topProducts} // This could be migrated to query later
            className="h-100" 
          />
        </Col>
      </Row>
    </>
  );
};

export default DefaultDashboardWithQuery;

// Example of how individual dashboard widgets could be updated to handle query states
const QueryAwareWidget = ({ data, isLoading, error, children, fallbackData = null }) => {
  if (isLoading) {
    return (
      <div className="d-flex justify-content-center align-items-center p-4">
        <Spinner animation="border" size="sm" />
      </div>
    );
  }

  if (error && !fallbackData) {
    return (
      <div className="text-center p-4">
        <small className="text-danger">Failed to load data</small>
      </div>
    );
  }

  // Use actual data if available, fallback data if error, or empty object
  const widgetData = data || fallbackData || {};
  
  return React.cloneElement(children, { data: widgetData });
};