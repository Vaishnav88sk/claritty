import React, { useState } from 'react';
import { CssBaseline, Box } from '@mui/material';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Dashboard } from './components/Dashboard';
import { KubernetesDashboard } from './components/KubernetesDashboard';
import { Navigation } from './components/Navigation';
import { ThemeProvider } from './theme/ThemeContext';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

export const App: React.FC = () => {
  const [view, setView] = useState<'system' | 'kubernetes'>('system');

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <CssBaseline />
        <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
        <Navigation view={view} onViewChange={setView} />
        {view === 'system' ? <Dashboard /> : <KubernetesDashboard />}
        </Box>
      </ThemeProvider>
    </QueryClientProvider>
  );
};
