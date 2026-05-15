import React from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { format } from 'date-fns';
import { Typography } from '@mui/material';
import type { MetricSeries } from '../types/metrics';

interface MetricChartProps {
  title: string;
  data: MetricSeries;
  color?: string;
  unit?: string;
  valueFormatter?: (value: number) => string;
}

export const MetricChart: React.FC<MetricChartProps> = ({
  title,
  data,
  color = '#8884d8',
  unit = '',
  valueFormatter = (value: number) => `${value}${unit}`,
}) => {
  const formatXAxis = (timestamp: string) => {
    return format(new Date(timestamp), 'HH:mm:ss');
  };

  const formatTooltip = (value: number) => {
    return valueFormatter(value);
  };

  return (
    <>
      <Typography variant="h6" gutterBottom color="white">
        {title}
      </Typography>
      <div style={{ width: '100%', height: 300 }}>
        <ResponsiveContainer>
          <LineChart
            data={data.data}
            margin={{
              top: 5,
              right: 30,
              left: 20,
              bottom: 5,
            }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#444" />
            <XAxis
              dataKey="timestamp"
              tickFormatter={formatXAxis}
              interval="preserveStartEnd"
              stroke="#888"
            />
            <YAxis
              tickFormatter={formatTooltip}
              stroke="#888"
            />
            <Tooltip
              labelFormatter={formatXAxis}
              formatter={formatTooltip}
              contentStyle={{
                backgroundColor: '#2a2a3e',
                border: 'none',
                color: 'white',
              }}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke={color}
              dot={false}
              name={title}
              strokeWidth={2}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </>
  );
}; 