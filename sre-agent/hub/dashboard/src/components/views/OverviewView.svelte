<script>
  import StatCard from '../StatCard.svelte';
  import ClusterCard from '../ClusterCard.svelte';
  import IncidentsTable from '../IncidentsTable.svelte';
  import EmptyState from '../EmptyState.svelte';
  import { formatMTTR } from '../../lib/utils.js';

  let {
    clusters = [],
    incidents = [],
    stats = {},
    watchTimers = {},
    onScan = () => {},
    onWatch = () => {},
    onStopWatch = () => {},
    onSelectIncident = () => {}
  } = $props();

  let openInc = $derived(
    incidents.filter((i) => i.status === 'INVESTIGATING').length
  );
  let sev1 = $derived(
    incidents.filter(
      (i) => i.severity === 'SEV1' && i.status === 'INVESTIGATING'
    ).length
  );
  let recentIncidents = $derived(incidents.slice(0, 10));
</script>

<div class="content" id="view-overview">
  <div class="stats-grid" id="global-stats">
    <StatCard label="Clusters" value={clusters.length} color="accent" />
    <StatCard
      label="Open Incidents"
      value={openInc}
      color={openInc > 0 ? 'red' : 'green'}
    />
    <StatCard
      label="SEV1 Active"
      value={sev1}
      color={sev1 > 0 ? 'red' : 'green'}
    />
    <StatCard
      label="Avg MTTR"
      value={formatMTTR(stats.avg_mttr_secs)}
      color="yellow"
    />
  </div>

  <h2 class="section-title">Clusters</h2>
  <div class="clusters-grid" id="clusters-overview">
    {#if clusters.length === 0}
      <EmptyState
        title="No clusters connected"
        subtitle="Deploy the agent to a cluster and point it to this hub."
      />
    {:else}
      {#each clusters as c (c.name)}
        {@const clusterIncCount = incidents.filter(
          (i) => i.cluster === c.name && i.status === 'INVESTIGATING'
        ).length}
        <ClusterCard
          cluster={c}
          openIncidentsCount={clusterIncCount}
          watchState={watchTimers[c.name]}
          {onScan}
          {onWatch}
          {onStopWatch}
        />
      {/each}
    {/if}
  </div>

  <h2 class="section-title">Recent Incidents</h2>
  <div id="recent-incidents-table">
    <IncidentsTable incidents={recentIncidents} onSelect={onSelectIncident} />
  </div>
</div>

