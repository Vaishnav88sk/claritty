import React from 'react';
import { Box } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { MetricCard } from './MetricCard';
import { ModernLayout } from './ModernLayout';
import { api } from '../services/api';

export const KubernetesDashboard: React.FC = () => {
  const { data: metrics, isLoading, error } = useQuery({
    queryKey: ['kubernetes-metrics'],
    queryFn: api.getKubernetesMetrics,
    refetchInterval: 10000,
  });

  // Example trends (you would calculate these from historical data)
  const nodeTrend = 0;  // Stable node count
  const podTrend = 2.5; // 2.5% increase in pods
  const cpuTrend = -1.2; // 1.2% decrease in CPU usage
  const memoryTrend = 3.8; // 3.8% increase in memory usage

  return (
    <ModernLayout
      title="Kubernetes Cluster Overview"
      isLoading={isLoading}
      error={error as Error}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        <Box sx={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(12, 1fr)', 
          gap: 3,
          '& > div': {
            transition: 'transform 0.3s ease, box-shadow 0.3s ease',
            '&:hover': {
              transform: 'translateY(-4px)',
              boxShadow: '0 8px 16px rgba(0,0,0,0.2)',
            },
          }
        }}>
          {/* Node Status */}
          <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
            <MetricCard
              title="Nodes"
              value={metrics?.nodes.ready || 0}
              maxValue={metrics?.nodes.total || 1}
              unit="/1"
              color="#6b4cf5"
              trend={nodeTrend}
              description="Available cluster nodes"
            />
          </Box>

          {/* Node CPU Usage */}
            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="Cluster CPU"
                value={metrics ? (metrics.resources.cpu.usage / metrics.resources.cpu.total) * 100 : 0}
                maxValue={100}
                unit="%"
                color="#4caf50"
                trend={cpuTrend}
                description="Total cluster CPU utilization"
              />
            </Box>

          {/* Node Memory Usage */}
            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="Cluster Memory"
                value={metrics ? (metrics.resources.memory.usage / metrics.resources.memory.total) * 100 : 0}
                maxValue={100}
                unit="%"
                color="#2196f3"
                trend={memoryTrend}
                description="Total cluster memory utilization"
              />
            </Box>

          {/* Pod Status */}
            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="Pods"
                value={metrics?.pods.running || 0}
                maxValue={metrics?.pods.total || 25}
                unit="/25"
                color="#ff4081"
                trend={podTrend}
                description="Running pods in cluster"
              />
            </Box>

          {/* Pod CPU Usage */}
            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="Pod CPU Usage"
                value={metrics ? (metrics.podResources.cpu.usage / metrics.podResources.cpu.total) * 100 : 0}
                maxValue={100}
                unit="%"
                color="#ff9800"
                description="Total pod CPU requests"
              />
            </Box>

          {/* Pod Memory Usage */}
            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="Pod Memory Usage"
                value={metrics ? (metrics.podResources.memory.usage / metrics.podResources.memory.total) * 100 : 0}
                maxValue={100}
                unit="%"
                color="#e91e63"
                description="Total pod memory requests"
              />
            </Box>

            {/* Resource Availability */}
            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="CPU Available"
                value={metrics ? ((metrics.resources.cpu.total - metrics.resources.cpu.usage) / metrics.resources.cpu.total) * 100 : 0}
                maxValue={100}
                unit="%"
                color="#009688"
                description="Available CPU resources"
              />
            </Box>

            <Box sx={{ gridColumn: { xs: 'span 12', md: 'span 6', lg: 'span 3' } }}>
              <MetricCard
                title="Memory Available"
                value={metrics ? ((metrics.resources.memory.total - metrics.resources.memory.usage) / metrics.resources.memory.total) * 100 : 0}
                maxValue={100}
                unit="%"
                color="#3f51b5"
                description="Available memory resources"
              />
            </Box>
        </Box>
      </Box>
    </ModernLayout>
  );
}; 