<script>
  import StatCard from '../StatCard.svelte';
  import { formatMTTR } from '../../lib/utils.js';

  let {
    incidents = [],
    stats = {}
  } = $props();

  let resolved = $derived(
    incidents.filter((i) => i.status === 'RESOLVED' || i.status === 'MITIGATED').length
  );
</script>

<div class="content" id="view-analytics">
  <div class="stats-grid" id="analytics-stats">
    <StatCard
      label="Total Incidents"
      value={stats.total_incidents || 0}
      color="accent"
    />
    <StatCard
      label="Open"
      value={stats.open_incidents || 0}
      color="yellow"
    />
    <StatCard
      label="Resolved / Mitigated"
      value={resolved}
      color="green"
    />
    <StatCard
      label="Avg MTTR"
      value={formatMTTR(stats.avg_mttr_secs)}
      color="yellow"
    />
  </div>
</div>

