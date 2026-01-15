import { Card, Spinner } from 'react-bootstrap';
import { LineChart } from 'echarts/charts';
import {
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import FalconCardHeader from 'components/common/FalconCardHeader';
import ReactEchart from 'components/common/ReactEchart';
import { useGetBillingStatsQuery } from 'store/api/billingApi';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faExclamationTriangle } from '@fortawesome/free-solid-svg-icons';
import { formatCurrency } from 'types/billing';

echarts.use([
  TooltipComponent,
  GridComponent,
  LineChart,
  CanvasRenderer,
  LegendComponent,
]);

const InvoiceTrendChart = () => {
  const { data: stats, isLoading, error } = useGetBillingStatsQuery({});

  const getChartOptions = () => {
    if (!stats) return {};

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
          label: {
            backgroundColor: '#6a7985',
          },
        },
        formatter: (params: any) => {
          let result = `<div class="fw-medium mb-1">${params[0].axisValue}</div>`;
          params.forEach((param: any) => {
            result += `<div class="d-flex align-items-center">
              <span class="badge rounded-circle p-1 me-2" style="background-color: ${param.color}"></span>
              <span>${param.seriesName}: ${formatCurrency(param.value)}</span>
            </div>`;
          });
          return result;
        },
      },
      legend: {
        data: ['Fatture Emesse', 'Fatture Ricevute'],
        bottom: 0,
        textStyle: {
          color: '#8991a7',
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '15%',
        top: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: ['Gen', 'Feb', 'Mar', 'Apr', 'Mag', 'Giu', 'Lug', 'Ago', 'Set', 'Ott', 'Nov', 'Dic'],
        axisLine: {
          lineStyle: {
            color: '#e6e6e6',
          },
        },
        axisLabel: {
          color: '#8991a7',
        },
      },
      yAxis: {
        type: 'value',
        axisLabel: {
          color: '#8991a7',
          formatter: (value: number) => {
            if (value >= 1000) {
              return `${(value / 1000).toFixed(0)}K`;
            }
            return value.toString();
          },
        },
        splitLine: {
          lineStyle: {
            color: '#f0f0f0',
          },
        },
      },
      series: [
        {
          name: 'Fatture Emesse',
          type: 'line',
          smooth: true,
          lineStyle: {
            width: 2,
          },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
                { offset: 1, color: 'rgba(59, 130, 246, 0.05)' },
              ],
            },
          },
          emphasis: {
            focus: 'series',
          },
          itemStyle: {
            color: '#3b82f6',
          },
          // Sample data - in real app this would come from API
          data: [
            stats.issuedAmount * 0.7,
            stats.issuedAmount * 0.8,
            stats.issuedAmount * 0.9,
            stats.issuedAmount * 0.75,
            stats.issuedAmount * 0.85,
            stats.issuedAmount * 0.95,
            stats.issuedAmount * 0.88,
            stats.issuedAmount * 0.6,
            stats.issuedAmount * 0.92,
            stats.issuedAmount * 0.98,
            stats.issuedAmount * 1.05,
            stats.issuedAmount,
          ].map((v) => Math.round(v)),
        },
        {
          name: 'Fatture Ricevute',
          type: 'line',
          smooth: true,
          lineStyle: {
            width: 2,
          },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
                { offset: 1, color: 'rgba(16, 185, 129, 0.05)' },
              ],
            },
          },
          emphasis: {
            focus: 'series',
          },
          itemStyle: {
            color: '#10b981',
          },
          // Sample data - in real app this would come from API
          data: [
            stats.receivedAmount * 0.65,
            stats.receivedAmount * 0.7,
            stats.receivedAmount * 0.82,
            stats.receivedAmount * 0.68,
            stats.receivedAmount * 0.78,
            stats.receivedAmount * 0.88,
            stats.receivedAmount * 0.75,
            stats.receivedAmount * 0.5,
            stats.receivedAmount * 0.85,
            stats.receivedAmount * 0.92,
            stats.receivedAmount * 0.98,
            stats.receivedAmount,
          ].map((v) => Math.round(v)),
        },
      ],
    };
  };

  if (isLoading) {
    return (
      <Card className="h-100">
        <FalconCardHeader title="Andamento Fatturazione" titleTag="h6" light />
        <Card.Body className="d-flex align-items-center justify-content-center" style={{ minHeight: 300 }}>
          <Spinner animation="border" />
        </Card.Body>
      </Card>
    );
  }

  if (error || !stats) {
    return (
      <Card className="h-100">
        <FalconCardHeader title="Andamento Fatturazione" titleTag="h6" light />
        <Card.Body className="d-flex align-items-center justify-content-center" style={{ minHeight: 300 }}>
          <div className="text-warning text-center">
            <FontAwesomeIcon icon={faExclamationTriangle} className="fs-3 mb-2 d-block mx-auto" />
            <span>Impossibile caricare il grafico</span>
          </div>
        </Card.Body>
      </Card>
    );
  }

  return (
    <Card className="h-100">
      <FalconCardHeader
        title="Andamento Fatturazione"
        titleTag="h6"
        light
        endEl={
          <div className="d-flex gap-3 fs-10">
            <div>
              <span className="text-body-tertiary">Emesse: </span>
              <span className="fw-medium text-primary">{formatCurrency(stats.issuedAmount)}</span>
            </div>
            <div>
              <span className="text-body-tertiary">Ricevute: </span>
              <span className="fw-medium text-success">{formatCurrency(stats.receivedAmount)}</span>
            </div>
          </div>
        }
      />
      <Card.Body>
        <ReactEchart
          echarts={echarts}
          option={getChartOptions()}
          style={{ height: 280 }}
        />
      </Card.Body>
    </Card>
  );
};

export default InvoiceTrendChart;
