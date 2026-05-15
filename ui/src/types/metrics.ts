export interface Metric {
  name: string;
  value: number;
  timestamp: string;
  tags?: Record<string, string>;
  unit?: string;
}

export interface MetricSeries {
  name: string;
  data: Array<{
    timestamp: string;
    value: number;
  }>;
}

export interface SystemMetrics {
  cpu: MetricSeries;
  memory: MetricSeries;
  disk: MetricSeries;
  network: {
    bytesReceived: MetricSeries;
    bytesSent: MetricSeries;
  };
  load: MetricSeries;
} 