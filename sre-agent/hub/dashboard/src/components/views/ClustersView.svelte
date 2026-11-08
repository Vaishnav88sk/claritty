<script>
  import ClusterCard from '../ClusterCard.svelte';
  import EmptyState from '../EmptyState.svelte';

  let {
    clusters = [],
    incidents = [],
    watchTimers = {},
    onScan = () => {},
    onWatch = () => {},
    onStopWatch = () => {}
  } = $props();
</script>

<div class="content" id="view-clusters">
  <div class="clusters-grid" id="clusters-detail">
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
          expanded={true}
          {onScan}
          {onWatch}
          {onStopWatch}
        />
      {/each}
    {/if}
  </div>
</div>

