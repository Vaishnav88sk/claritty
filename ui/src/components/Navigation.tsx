import React from 'react';
import { 
  AppBar, 
  Toolbar, 
  Typography, 
  Box, 
  ToggleButtonGroup, 
  ToggleButton, 
  IconButton,
  Tooltip,
} from '@mui/material';
import { 
  Computer, 
  Storage,
  LightMode,
  DarkMode,
} from '@mui/icons-material';
import { useTheme } from '../theme/ThemeContext';

interface NavigationProps {
  view: 'system' | 'kubernetes';
  onViewChange: (view: 'system' | 'kubernetes') => void;
}

export const Navigation: React.FC<NavigationProps> = ({ view, onViewChange }) => {
  const { isDarkMode, toggleTheme } = useTheme();

  const handleChange = (
    _event: React.MouseEvent<HTMLElement>,
    newView: 'system' | 'kubernetes' | null
  ) => {
    if (newView !== null) {
      onViewChange(newView);
    }
  };

  return (
    <AppBar 
      position="fixed" 
      sx={{ 
        backgroundColor: isDarkMode ? '#1a1a2e' : '#ffffff',
        boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
        backdropFilter: 'blur(10px)',
        background: isDarkMode 
          ? 'linear-gradient(180deg, rgba(26,26,46,0.95) 0%, rgba(26,26,46,0.9) 100%)'
          : 'linear-gradient(180deg, rgba(255,255,255,0.95) 0%, rgba(255,255,255,0.9) 100%)',
      }}
    >
      <Toolbar>
        <Typography 
          variant="h6" 
          component="div" 
          sx={{ 
            flexGrow: 1, 
            fontWeight: 600,
            letterSpacing: '0.5px',
            color: isDarkMode ? '#fff' : '#2c3e50',
          }}
        >
          Claritty Dashboard
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <ToggleButtonGroup
            value={view}
            exclusive
            onChange={handleChange}
            aria-label="view selector"
            sx={{
              backgroundColor: isDarkMode ? 'rgba(42, 42, 62, 0.5)' : 'rgba(245, 245, 245, 0.5)',
              padding: '4px',
              borderRadius: '12px',
              border: isDarkMode ? '1px solid rgba(255, 255, 255, 0.1)' : '1px solid rgba(0, 0, 0, 0.1)',
              backdropFilter: 'blur(8px)',
              '& .MuiToggleButton-root': {
                color: isDarkMode ? '#fff' : '#2c3e50',
                padding: '8px 24px',
                borderRadius: '8px',
                transition: 'all 0.3s ease',
                border: 'none',
                margin: '0 4px',
                '&.Mui-selected': {
                  backgroundColor: '#3f51b5',
                  color: '#fff',
                  boxShadow: '0 4px 12px rgba(63, 81, 181, 0.3)',
                  '&:hover': {
                    backgroundColor: '#3949ab',
                  },
                },
                '&:hover': {
                  backgroundColor: isDarkMode ? 'rgba(63, 81, 181, 0.1)' : 'rgba(63, 81, 181, 0.05)',
                  transform: 'translateY(-1px)',
                },
              },
            }}
          >
            <ToggleButton value="system" aria-label="system metrics">
              <Computer sx={{ mr: 1 }} />
              System
            </ToggleButton>
            <ToggleButton value="kubernetes" aria-label="kubernetes metrics">
              <Storage sx={{ mr: 1 }} />
              Kubernetes
            </ToggleButton>
          </ToggleButtonGroup>
          <Tooltip title={`Switch to ${isDarkMode ? 'Light' : 'Dark'} Mode`}>
            <IconButton 
              onClick={toggleTheme}
              sx={{ 
                color: isDarkMode ? '#fff' : '#2c3e50',
                backgroundColor: isDarkMode ? 'rgba(42, 42, 62, 0.5)' : 'rgba(245, 245, 245, 0.5)',
                borderRadius: '12px',
                padding: '8px',
                transition: 'all 0.3s ease',
                '&:hover': {
                  backgroundColor: isDarkMode ? 'rgba(63, 81, 181, 0.1)' : 'rgba(63, 81, 181, 0.05)',
                  transform: 'translateY(-1px)',
                },
              }}
            >
              {isDarkMode ? <LightMode /> : <DarkMode />}
            </IconButton>
          </Tooltip>
        </Box>
      </Toolbar>
    </AppBar>
  );
}; 