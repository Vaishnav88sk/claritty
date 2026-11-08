<script>
  import { formatCountdown } from '../lib/utils.js';

  let {
    cluster,
    openIncidentsCount = 0,
    watchState = null,
    onScan = () => {},
    onWatch = () => {},
    onStopWatch = () => {}
  } = $props();

  let score = $derived(Math.round(cluster.health_score || 0));
  let healthClass = $derived(
    score >= 80 ? 'health-good' : score >= 50 ? 'health-warn' : 'health-crit'
  );
  let namespaces = $derived((cluster.namespaces || []).slice(0, 6));
</script>

<div class="cluster-card">
  <div class="cluster-header">
    <div class="cluster-name">{cluster.name}</div>
    <div class="health-badge {healthClass}">● {score}/100</div>
  </div>
  <div class="cluster-stats">
    <div class="cluster-stat">
      Nodes <span>{cluster.ready_nodes || 0}/{cluster.total_nodes || 0} Ready</span>
    </div>
    <div class="cluster-stat">
      Pods Running <span>{cluster.running_pods || 0}</span>
    </div>
    <div class="cluster-stat">
      Pending <span>{cluster.pending_pods || 0}</span>
    </div>
    <div class="cluster-stat">
      CrashLoop
      <span style="color:var(--{cluster.crashloop > 0 ? 'red' : 'text'})">
        {cluster.crashloop || 0}
      </span>
    </div>
  </div>

  {#if namespaces.length > 0}
    <div class="cluster-namespaces">
      {#each namespaces as ns}
        <span class="ns-badge">{ns}</span>
      {/each}
    </div>
  {/if}

  {#if openIncidentsCount > 0}
    <div style="font-size:12px;color:var(--red);margin:8px 0">
      ⚠ {openIncidentsCount} open incident{openIncidentsCount > 1 ? 's' : ''}
    </div>
  {/if}

  <div class="cluster-actions">
    <button class="btn btn-ghost btn-sm" onclick={() => onScan(cluster.name)}>
      🔍 Scan Once
    </button>
    {#if watchState && watchState.running}
      <button class="btn btn-danger btn-sm" onclick={() => onStopWatch(cluster.name)}>
        ⏸ Stop Watching
      </button>
    {:else}
      <button class="btn btn-accent btn-sm" onclick={() => onWatch(cluster.name)}>
        ▶ Watch
      </button>
    {/if}
  </div>

  {#if watchState && watchState.running}
    <div class="watch-info">
      Next scan in {formatCountdown(watchState.remaining)}
    </div>
  {/if}
</div>

