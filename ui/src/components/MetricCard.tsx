import React from 'react';
import { Card, CardContent, Typography, Box, LinearProgress, Tooltip } from '@mui/material';
import { styled } from '@mui/material/styles';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';

const StyledCard = styled(Card)(({ theme }) => ({
  height: '100%',
  display: 'flex',
  flexDirection: 'column',
  position: 'relative',
  overflow: 'visible',
  '&::before': {
    content: '""',
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: 4,
    backgroundColor: theme.palette.primary.main,
    borderTopLeftRadius: theme.shape.borderRadius,
    borderTopRightRadius: theme.shape.borderRadius,
  },
}));

const MetricValue = styled(Typography)(({ theme }) => ({
  fontSize: '2rem',
  fontWeight: 600,
  color: theme.palette.text.primary,
  marginRight: theme.spacing(1),
}));

const MetricUnit = styled(Typography)(({ theme }) => ({
  fontSize: '0.875rem',
  color: theme.palette.text.secondary,
  marginTop: 'auto',
}));

interface MetricCardProps {
  title: string;
  value: number;
  maxValue: number;
  unit: string;
  color?: string;
  trend?: number;
  description?: string;
}

export const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  maxValue,
  unit,
  color = '#2196f3',
  trend,
  description,
}) => {
  const percentage = (value / maxValue) * 100;
  const formattedValue = value.toFixed(1);
  
  const getTrendColor = (trend: number) => {
    if (trend > 0) return '#4caf50';
    if (trend < 0) return '#f44336';
    return '#757575';
  };

  return (
    <StyledCard>
      <CardContent>
        <Box mb={2}>
          <Typography variant="subtitle2" color="textSecondary" gutterBottom>
            {title}
          </Typography>
          <Box display="flex" alignItems="flex-end" mb={1}>
            <MetricValue>{formattedValue}</MetricValue>
            <MetricUnit>{unit}</MetricUnit>
            {trend !== undefined && (
              <Box ml="auto" display="flex" alignItems="center">
                <Tooltip title={`${Math.abs(trend)}% ${trend >= 0 ? 'increase' : 'decrease'}`}>
                  <Box display="flex" alignItems="center" sx={{ color: getTrendColor(trend) }}>
                    {trend > 0 ? <TrendingUpIcon /> : <TrendingDownIcon />}
                    <Typography variant="caption" sx={{ ml: 0.5 }}>
                      {Math.abs(trend)}%
                    </Typography>
                  </Box>
                </Tooltip>
              </Box>
            )}
          </Box>
        </Box>
        
        <Box>
          <LinearProgress
            variant="determinate"
            value={Math.min(percentage, 100)}
            sx={{
              height: 8,
              borderRadius: 4,
              backgroundColor: 'rgba(0,0,0,0.05)',
              '& .MuiLinearProgress-bar': {
                backgroundColor: color,
                borderRadius: 4,
              },
            }}
          />
          {description && (
            <Typography variant="caption" color="textSecondary" sx={{ mt: 1, display: 'block' }}>
              {description}
            </Typography>
          )}
        </Box>
      </CardContent>
    </StyledCard>
  );
}; 