<script>
  let { currentView = 'overview', hubStatus = false, onNavigate = () => {} } = $props();

  function handleClick(e, view) {
    e.preventDefault();
    onNavigate(view);
  }
</script>

<aside class="sidebar" id="sidebar">
  <div class="sidebar-brand">
    <span class="brand-icon">◈</span>
    <span class="brand-name">Claritty</span>
    <span class="brand-tag">AI-SRE</span>
  </div>
  <nav class="sidebar-nav">
    <a
      href="#overview"
      class="nav-item {currentView === 'overview' ? 'active' : ''}"
      data-view="overview"
      onclick={(e) => handleClick(e, 'overview')}
    >
      <span class="nav-icon">⬡</span> Overview
    </a>
    <a
      href="#clusters"
      class="nav-item {currentView === 'clusters' ? 'active' : ''}"
      data-view="clusters"
      onclick={(e) => handleClick(e, 'clusters')}
    >
      <span class="nav-icon">⬡</span> Clusters
    </a>
    <div class="nav-divider"></div>
    <a
      href="#incidents"
      class="nav-item {currentView === 'incidents' ? 'active' : ''}"
      data-view="incidents"
      onclick={(e) => handleClick(e, 'incidents')}
    >
      <span class="nav-icon">⬡</span> All Incidents
    </a>
    <a
      href="#analytics"
      class="nav-item {currentView === 'analytics' ? 'active' : ''}"
      data-view="analytics"
      onclick={(e) => handleClick(e, 'analytics')}
    >
      <span class="nav-icon">⬡</span> Analytics
    </a>
  </nav>
  <div class="sidebar-footer">
    <span class="status-dot {hubStatus ? 'ok' : 'error'}" id="hub-status-dot"></span>
    <span id="hub-status-text">{hubStatus ? 'Hub Connected' : 'Hub Offline'}</span>
  </div>
</aside>

