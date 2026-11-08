<script>
  let { title = 'Overview', lastRefresh = '—', onRefresh = () => {} } = $props();
</script>

<header class="topbar">
  <div class="topbar-title" id="page-title">{title}</div>
  <div class="topbar-actions">
    <span class="last-refresh" id="last-refresh">{lastRefresh}</span>
    <button class="btn btn-ghost" onclick={onRefresh}>↻ Refresh</button>
  </div>
</header>

