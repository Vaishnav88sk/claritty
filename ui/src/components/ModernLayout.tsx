import React from 'react';
import {
  Box,
  Container,
  Typography,
  Skeleton,
  Alert,
} from '@mui/material';
import { useTheme } from '../theme/ThemeContext';

interface ModernLayoutProps {
  title: string;
  isLoading?: boolean;
  error?: Error | null;
  children?: React.ReactNode;
}

export const ModernLayout: React.FC<ModernLayoutProps> = ({
  title,
  isLoading,
  error,
  children,
}) => {
  const { isDarkMode } = useTheme();

  return (
    <Box
      component="main"
      sx={{
        flexGrow: 1,
        p: 3,
        width: '100%',
        mt: '64px',
        minHeight: 'calc(100vh - 64px)',
        background: isDarkMode 
          ? 'linear-gradient(180deg, #1a1a2e 0%, #16162a 100%)'
          : 'linear-gradient(180deg, #f8f9fc 0%, #f1f4f9 100%)',
        transition: 'all 0.3s ease',
      }}
    >
      <Container 
        maxWidth="xl" 
        sx={{
          py: 4,
          opacity: isLoading ? 0.7 : 1,
          transition: 'opacity 0.3s ease',
        }}
      >
        <Typography
          variant="h4"
          gutterBottom
          sx={{
            mb: 4,
            fontWeight: 600,
            color: isDarkMode ? '#fff' : '#2c3e50',
            letterSpacing: '0.5px',
            transform: isLoading ? 'scale(0.98)' : 'scale(1)',
            transition: 'transform 0.3s ease',
          }}
        >
          {title}
        </Typography>

        {isLoading && (
          <Box 
            sx={{ 
              py: 4,
              '& .MuiSkeleton-root': {
                transform: 'scale(0.98)',
                transition: 'transform 0.3s ease, opacity 0.3s ease',
                '&:hover': {
                  transform: 'scale(0.99)',
                  opacity: 0.8,
                },
              },
            }}
          >
            <Skeleton 
              variant="rectangular" 
              height={200} 
              sx={{ 
                borderRadius: 2, 
                mb: 2,
                background: isDarkMode
                  ? 'linear-gradient(90deg, #2a2a3e 0%, #1e1e2d 100%)'
                  : 'linear-gradient(90deg, #f1f4f9 0%, #e8ecf1 100%)',
              }} 
            />
            <Skeleton 
              variant="rectangular" 
              height={200} 
              sx={{ 
                borderRadius: 2,
                background: isDarkMode
                  ? 'linear-gradient(90deg, #2a2a3e 0%, #1e1e2d 100%)'
                  : 'linear-gradient(90deg, #f1f4f9 0%, #e8ecf1 100%)',
              }} 
            />
          </Box>
        )}

        {error && (
          <Alert 
            severity="error" 
            sx={{ 
              mb: 4,
              borderRadius: 2,
              backdropFilter: 'blur(8px)',
              background: isDarkMode
                ? 'rgba(211, 47, 47, 0.1)'
                : 'rgba(211, 47, 47, 0.05)',
              border: isDarkMode
                ? '1px solid rgba(211, 47, 47, 0.2)'
                : '1px solid rgba(211, 47, 47, 0.1)',
              color: isDarkMode ? '#fff' : '#d32f2f',
              '& .MuiAlert-icon': {
                color: '#f44336',
              },
            }}
          >
            {error.message}
          </Alert>
        )}

        <Box 
          sx={{
            opacity: isLoading ? 0 : 1,
            transform: isLoading ? 'translateY(10px)' : 'translateY(0)',
            transition: 'opacity 0.3s ease, transform 0.3s ease',
          }}
        >
          {!isLoading && !error && children}
        </Box>
      </Container>
    </Box>
  );
}; 