import React from 'react';
import { Box } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { ModernLayout } from './ModernLayout';
import { MetricCard } from './MetricCard';
import { api } from '../services/api';
import type { Metric, MetricSeries } from '../types/metrics';

const processMetrics = (metrics: Metric[]): Record<string, MetricSeries> => {
  const result: Record<string, MetricSeries> = {};

  metrics.forEach((metric) => {
    if (!result[metric.name]) {
      result[metric.name] = {
        name: metric.name,
        data: [],
      };
    }
    result[metric.name].data.push({
      timestamp: metric.timestamp,
      value: metric.value,
    });
  });

  return result;
};

export const Dashboard: React.FC = () => {
  const { data: metrics, isLoading, error } = useQuery({
    queryKey: ['metrics'],
    queryFn: api.getMetricsSnapshot,
    refetchInterval: 10000,
  });

  const processedMetrics = metrics ? processMetrics(metrics) : null;
  const latestCpuUsage = processedMetrics?.['cpu.usage']?.data.slice(-1)[0]?.value || 0;
  const latestMemoryUsage = processedMetrics?.['memory.usage_percent']?.data.slice(-1)[0]?.value || 0;
  const latestDiskUsed = processedMetrics?.['disk.used']?.data.slice(-1)[0]?.value || 0;
  const latestDiskTotal = latestDiskUsed + (processedMetrics?.['disk.free']?.data.slice(-1)[0]?.value || 0);
  const diskUsagePercent = (latestDiskUsed / latestDiskTotal) * 100;
  const latestNetworkRecv = processedMetrics?.['network.bytes_recv_per_sec']?.data.slice(-1)[0]?.value || 0;
  const latestNetworkSent = processedMetrics?.['network.bytes_sent_per_sec']?.data.slice(-1)[0]?.value || 0;
  const latestLoad1 = processedMetrics?.['load.1min']?.data.slice(-1)[0]?.value || 0;
  const latestLoad5 = processedMetrics?.['load.5min']?.data.slice(-1)[0]?.value || 0;

  // Calculate trends (example - you would need historical data for real trends)
  const cpuTrend = 5.2;  // Example trend
  const memoryTrend = -2.1;
  const diskTrend = 1.5;
  const networkTrend = 3.8;

  return (
    <ModernLayout
      title="System Overview"
      isLoading={isLoading}
      error={error as Error}
    >
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(12, 1fr)', gap: 3 }}>
          {/* Resource Usage */}
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
                title="CPU Usage"
                value={latestCpuUsage}
                maxValue={100}
                unit="%"
            color="#2196f3"
            trend={cpuTrend}
            description="Total CPU utilization across all cores"
              />
        </Box>
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
                title="Memory Usage"
                value={latestMemoryUsage}
                maxValue={100}
                unit="%"
            color="#4caf50"
            trend={memoryTrend}
            description="Physical memory utilization"
              />
        </Box>
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
                title="Disk Usage"
                value={diskUsagePercent}
                maxValue={100}
                unit="%"
                color="#ff9800"
            trend={diskTrend}
            description="Storage space utilization"
          />
        </Box>
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
            title="Network I/O"
            value={latestNetworkRecv + latestNetworkSent}
            maxValue={Math.max((latestNetworkRecv + latestNetworkSent) * 1.2, 100)}
                unit="KB/s"
                color="#9c27b0"
            trend={networkTrend}
            description="Combined network throughput"
              />
        </Box>

          {/* System Load */}
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
            title="System Load (1m)"
                value={latestLoad1}
            maxValue={Math.max(latestLoad5, latestLoad1, 1)}
                unit=""
                color="#f44336"
            description="Average system load over 1 minute"
              />
        </Box>
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
            title="System Load (5m)"
            value={latestLoad5}
            maxValue={Math.max(latestLoad5, latestLoad1, 1)}
            unit=""
            color="#e91e63"
            description="Average system load over 5 minutes"
          />
        </Box>
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
            title="Network In"
            value={latestNetworkRecv}
            maxValue={Math.max(latestNetworkRecv * 1.2, 100)}
            unit="KB/s"
            color="#673ab7"
            description="Incoming network traffic"
          />
        </Box>
        <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
          <MetricCard
            title="Network Out"
            value={latestNetworkSent}
            maxValue={Math.max(latestNetworkSent * 1.2, 100)}
            unit="KB/s"
            color="#3f51b5"
            description="Outgoing network traffic"
          />
        </Box>
    </Box>
    </ModernLayout>
  );
}; 