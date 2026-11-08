<script>
  import IncidentsTable from '../IncidentsTable.svelte';

  let {
    clusters = [],
    incidents = [],
    onSelectIncident = () => {}
  } = $props();

  let filterCluster = $state('');
  let filterSeverity = $state('');
  let filterStatus = $state('');

  let filteredIncidents = $derived(
    incidents.filter((i) => {
      if (filterCluster && i.cluster !== filterCluster) return false;
      if (filterSeverity && i.severity !== filterSeverity) return false;
      if (filterStatus && i.status !== filterStatus) return false;
      return true;
    })
  );
</script>

<div class="content" id="view-incidents">
  <div class="filter-bar">
    <select id="filter-cluster" bind:value={filterCluster}>
      <option value="">All Clusters</option>
      {#each clusters as c}
        <option value={c.name}>{c.name}</option>
      {/each}
    </select>

    <select id="filter-severity" bind:value={filterSeverity}>
      <option value="">All Severities</option>
      <option value="SEV1">SEV1</option>
      <option value="SEV2">SEV2</option>
      <option value="SEV3">SEV3</option>
    </select>

    <select id="filter-status" bind:value={filterStatus}>
      <option value="">All Statuses</option>
      <option value="INVESTIGATING">Investigating</option>
      <option value="MITIGATED">Mitigated</option>
      <option value="RESOLVED">Resolved</option>
    </select>
  </div>

  <div id="all-incidents-table">
    <IncidentsTable incidents={filteredIncidents} onSelect={onSelectIncident} />
  </div>
</div>

