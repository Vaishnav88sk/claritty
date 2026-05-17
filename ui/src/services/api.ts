import axios from 'axios';
import type { Metric } from '../types/metrics';

// Use environment variable for API base URL, fallback to dev default
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || (window.location.hostname === 'localhost' ? 'http://localhost:8088/api' : '/api');

interface MetricsResponse {
  timestamp: string;
  metrics: {
    cpu: Record<string, MetricData>;
    memory: Record<string, MetricData>;
    disk: Record<string, MetricData>;
    network: Record<string, MetricData>;
    load: Record<string, MetricData>;
  };
}

// interface KubernetesMetricsResponse {
//   timestamp: string;
//   metrics: {
//     nodes: {
//       total: number;
//       ready: number;
//     };
//     pods: {
//       total: number;
//       running: number;
//     };
//     resources: {
//       cpu: {
//         usage: number;
//         total: number;
//       };
//       memory: {
//         usage: number;
//         total: number;
//       };
//     };
//     podResources: {
//       cpu: {
//         usage: number;
//         total: number;
//       };
//       memory: {
//         usage: number;
//         total: number;
//       };
//     };
//   };
// }

interface KubernetesMetricsResponse {
  nodes: {
    total: number;
    ready: number;
  };
  pods: {
    total: number;
    running: number;
  };
  resources: {
    cpu: {
      usage: number;
      total: number;
    };
    memory: {
      usage: number;
      total: number;
    };
  };
  podResources: {
    cpu: {
      usage: number;
      total: number;
    };
    memory: {
      usage: number;
      total: number;
    };
  };
}

interface MetricData {
  value: number;
  unit?: string;
  tags?: Record<string, string>;
  timestamp: string;
}

const getMetrics = async (): Promise<Metric[]> => {
  const response = await axios.get<MetricsResponse>(`${API_BASE_URL}/metrics/snapshot`);
  const metrics: Metric[] = [];

  Object.entries(response.data.metrics).forEach(([category, categoryMetrics]) => {
    Object.entries(categoryMetrics).forEach(([name, data]) => {
      metrics.push({
        name: `${category}.${name}`,
        value: data.value,
        timestamp: data.timestamp,
        unit: data.unit,
        tags: data.tags
      });
    });
  });

  return metrics;
};

const getKubernetesMetrics = async (): Promise<KubernetesMetricsResponse> => {
  // const response = await axios.get<KubernetesMetricsResponse>(`${API_BASE_URL}/kubernetes/metrics`);
  const response = await axios.get(`${API_BASE_URL}/cluster/overview`);
  // return response.data.metrics;
  return response.data;
};

export const api = {
  getMetricsSnapshot: getMetrics,
  getKubernetesMetrics
}; 