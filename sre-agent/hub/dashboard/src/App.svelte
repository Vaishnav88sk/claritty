<script>
  import { onMount, onDestroy } from 'svelte';
  import Sidebar from './components/Sidebar.svelte';
  import Topbar from './components/Topbar.svelte';
  import Toast from './components/Toast.svelte';
  import IncidentModal from './components/IncidentModal.svelte';

  import OverviewView from './components/views/OverviewView.svelte';
  import ClustersView from './components/views/ClustersView.svelte';
  import IncidentsView from './components/views/IncidentsView.svelte';
  import AnalyticsView from './components/views/AnalyticsView.svelte';

  import * as api from './lib/api.js';

  const POLL_INTERVAL = 30000;

  let clusters = $state([]);
  let incidents = $state([]);
  let stats = $state({});
  let hubStatus = $state(false);
  let lastRefresh = $state('—');
  let currentView = $state('overview');
  let watchTimers = $state({});

  // Modal state
  let modalOpen = $state(false);
  let activeIncident = $state(null);
  let loadingIncident = $state(false);
  let incidentError = $state(null);

  // Toast state
  let toastMessage = $state('');
  let toastColor = $state('');
  let toastVisible = $state(false);
  let toastTimeout = null;

  let pollTimer = null;

  const titles = {
    overview: 'Overview',
    clusters: 'Clusters',
    incidents: 'Incidents',
    analytics: 'Analytics'
  };

  let pageTitle = $derived(titles[currentView] || currentView);

  function showToast(msg, color = '') {
    toastMessage = msg;
    toastColor = color;
    toastVisible = true;
    if (toastTimeout) clearTimeout(toastTimeout);
    toastTimeout = setTimeout(() => {
      toastVisible = false;
    }, 3000);
  }

  function copyCmd(cmd) {
    if (navigator?.clipboard?.writeText) {
      navigator.clipboard.writeText(cmd).then(() => {
        showToast('Command copied!', 'green');
      });
    }
  }

  async function fetchAll() {
    try {
      const [cData, iData, sData] = await Promise.all([
        api.fetchClusters(),
        api.fetchIncidents(200),
        api.fetchStats()
      ]);
      clusters = cData || [];
      incidents = iData || [];
      stats = sData || {};
      hubStatus = true;
      lastRefresh = 'Updated ' + new Date().toLocaleTimeString();
    } catch (e) {
      hubStatus = false;
      console.error('Fetch error:', e);
    }
  }

  async function triggerScan(clusterName) {
    showToast(`Triggering scan for ${clusterName}…`);
    try {
      const res = await api.triggerScan(clusterName);
      if (res.ok) {
        showToast(`✓ Scan started for ${clusterName}`, 'green');
        setTimeout(fetchAll, 5000);
      } else {
        showToast('Scan trigger failed — check agent connectivity', 'red');
      }
    } catch {
      showToast('Cannot reach hub API', 'red');
    }
  }

  async function startWatch(clusterName) {
    try {
      await api.startWatch(clusterName);
      const interval = setInterval(() => {
        if (!watchTimers[clusterName]) return;
        watchTimers[clusterName].remaining--;
        if (watchTimers[clusterName].remaining <= 0) {
          watchTimers[clusterName].remaining = 300;
          fetchAll();
        }
      }, 1000);

      watchTimers[clusterName] = {
        running: true,
        remaining: 300,
        interval
      };
      showToast(`▶ Watching ${clusterName}`, 'green');
    } catch {
      showToast('Failed to start watch', 'red');
    }
  }

  async function stopWatch(clusterName) {
    try {
      await api.stopWatch(clusterName);
      if (watchTimers[clusterName]?.interval) {
        clearInterval(watchTimers[clusterName].interval);
      }
      const updated = { ...watchTimers };
      delete updated[clusterName];
      watchTimers = updated;
      showToast(`⏸ Watch stopped for ${clusterName}`, '');
    } catch {
      showToast('Failed to stop watch', 'red');
    }
  }

  async function openIncident(id) {
    modalOpen = true;
    activeIncident = null;
    loadingIncident = true;
    incidentError = null;

    try {
      const data = await api.fetchIncident(id);
      activeIncident = data;
    } catch {
      incidentError = 'Failed to load incident';
    } finally {
      loadingIncident = false;
    }
  }

  function closeModal() {
    modalOpen = false;
    activeIncident = null;
    loadingIncident = false;
    incidentError = null;
  }

  async function updateStatus(id, status) {
    try {
      await api.updateIncidentStatus(id, status);
      closeModal();
      await fetchAll();
      showToast(`Incident marked as ${status}`, 'green');
    } catch (e) {
      showToast('Failed to update status', 'red');
    }
  }

  function handleNavigate(view) {
    currentView = view;
  }

  onMount(() => {
    fetchAll();
    pollTimer = setInterval(fetchAll, POLL_INTERVAL);
  });

  onDestroy(() => {
    if (pollTimer) clearInterval(pollTimer);
    if (toastTimeout) clearTimeout(toastTimeout);
    Object.values(watchTimers).forEach((t) => {
      if (t?.interval) clearInterval(t.interval);
    });
  });
</script>

<Sidebar
  {currentView}
  {hubStatus}
  onNavigate={handleNavigate}
/>

<main class="main">
  <Topbar
    title={pageTitle}
    {lastRefresh}
    onRefresh={fetchAll}
  />

  {#if currentView === 'overview'}
    <OverviewView
      {clusters}
      {incidents}
      {stats}
      {watchTimers}
      onScan={triggerScan}
      onWatch={startWatch}
      onStopWatch={stopWatch}
      onSelectIncident={openIncident}
    />
  {:else if currentView === 'clusters'}
    <ClustersView
      {clusters}
      {incidents}
      {watchTimers}
      onScan={triggerScan}
      onWatch={startWatch}
      onStopWatch={stopWatch}
    />
  {:else if currentView === 'incidents'}
    <IncidentsView
      {clusters}
      {incidents}
      onSelectIncident={openIncident}
    />
  {:else if currentView === 'analytics'}
    <AnalyticsView
      {incidents}
      {stats}
    />
  {/if}
</main>

{#if modalOpen}
  <IncidentModal
    incident={activeIncident}
    loading={loadingIncident}
    error={incidentError}
    onClose={closeModal}
    onUpdateStatus={updateStatus}
    onCopyCmd={copyCmd}
  />
{/if}

<Toast
  message={toastMessage}
  color={toastColor}
  visible={toastVisible}
/>

