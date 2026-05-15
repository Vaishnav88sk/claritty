import React from 'react';
import { Container, Typography, Box } from '@mui/material';

interface DashboardLayoutProps {
  title: string;
  isLoading?: boolean;
  error?: Error | null;
  children?: React.ReactNode;
}

export const DashboardLayout: React.FC<DashboardLayoutProps> = ({
  title,
  isLoading,
  error,
  children,
}) => {
  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="calc(100vh - 64px)">
        <Typography variant="h5" color="white">Loading {title.toLowerCase()}...</Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="calc(100vh - 64px)">
        <Typography color="error" variant="h5">Error loading {title.toLowerCase()}: {error.message}</Typography>
      </Box>
    );
  }

  if (!children) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="calc(100vh - 64px)">
        <Typography variant="h5" color="white">No metrics available</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ backgroundColor: '#1e1e2d', minHeight: 'calc(100vh - 64px)', py: 4 }}>
      <Container maxWidth="xl">
        <Typography variant="h5" gutterBottom sx={{ color: '#ffffff', mb: 4 }}>
          {title}
        </Typography>
        {children}
      </Container>
    </Box>
  );
}; 