import React from 'react';
import { Box, Typography } from '@mui/material';
import { PieChart, Pie, Cell } from 'recharts';

interface GaugeChartProps {
  title: string;
  value: number;
  maxValue: number;
  unit: string;
  color: string;
}

export const GaugeChart: React.FC<GaugeChartProps> = ({
  title,
  value,
  maxValue,
  unit,
  color,
}) => {
  const normalizedValue = Math.min(Math.max(value, 0), maxValue);
  const percentage = (normalizedValue / maxValue) * 100;
  
  const data = [
    { name: 'value', value: percentage },
    { name: 'empty', value: 100 - percentage },
  ];

  return (
    <Box sx={{ textAlign: 'center' }}>
      <Typography variant="h6" gutterBottom color="white">
        {title}
      </Typography>
      <Box sx={{ position: 'relative', display: 'inline-flex' }}>
        <PieChart width={200} height={100}>
          <Pie
            data={data}
            cx={100}
            cy={100}
            startAngle={180}
            endAngle={0}
            innerRadius={60}
            outerRadius={80}
            paddingAngle={0}
            dataKey="value"
          >
            <Cell fill={color} />
            <Cell fill="#444" />
          </Pie>
        </PieChart>
        <Box
          sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -20%)',
            textAlign: 'center',
          }}
        >
          <Typography variant="h4" component="div" color="white">
            {percentage.toFixed(1)}
            <Typography variant="caption" component="span" color="white">
              {unit}
            </Typography>
          </Typography>
        </Box>
      </Box>
    </Box>
  );
}; 