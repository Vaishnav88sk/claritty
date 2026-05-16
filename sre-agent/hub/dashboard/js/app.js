// Claritty SRE Dashboard — Vanilla JS SPA
// Polls /api/v1/* every 30s and updates DOM in place.

const API = '';
const POLL_INTERVAL = 30000;

// ─── State ────────────────────────────────────────────────────────────────────
let state = {
  clusters: [],
  incidents: [],
  stats: {},
  watchTimers: {}, // clusterName → { timer, remaining }
  currentView: 'overview',
};

// ─── Navigation ───────────────────────────────────────────────────────────────
function navigate(view) {
  state.currentView = view;
  document.querySelectorAll('.content').forEach(el => el.classList.add('hidden'));
  document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));

  const viewEl = document.getElementById(`view-${view}`);
  if (viewEl) viewEl.classList.remove('hidden');

  const navEl = document.querySelector(`[data-view="${view}"]`);
  if (navEl) navEl.classList.add('active');

  const titles = { overview: 'Overview', clusters: 'Clusters', incidents: 'Incidents', analytics: 'Analytics' };
  document.getElementById('page-title').textContent = titles[view] || view;

  renderCurrentView();
}

// ─── Data Fetching ────────────────────────────────────────────────────────────
async function fetchAll() {
  try {
    const [clusters, incidents, stats] = await Promise.all([
      fetch(`${API}/api/v1/clusters`).then(r => r.json()),
      fetch(`${API}/api/v1/incidents?limit=200`).then(r => r.json()),
      fetch(`${API}/api/v1/stats`).then(r => r.json()),
    ]);
    state.clusters = clusters || [];
    state.incidents = incidents || [];
    state.stats = stats || {};

    setHubStatus(true);
    document.getElementById('last-refresh').textContent = 'Updated ' + new Date().toLocaleTimeString();
    populateClusterFilter();
    renderCurrentView();
  } catch (e) {
    setHubStatus(false);
    console.error('Fetch error:', e);
  }
}

function setHubStatus(ok) {
  const dot = document.getElementById('hub-status-dot');
  const txt = document.getElementById('hub-status-text');
  dot.className = 'status-dot ' + (ok ? 'ok' : 'error');
  txt.textContent = ok ? 'Hub Connected' : 'Hub Offline';
}

async function refreshAll() {
  await fetchAll();
}

// ─── Render Dispatcher ────────────────────────────────────────────────────────
function renderCurrentView() {
  switch (state.currentView) {
    case 'overview':   renderOverview(); break;
    case 'clusters':   renderClustersView(); break;
    case 'incidents':  renderAllIncidents(); break;
    case 'analytics':  renderAnalytics(); break;
  }
}

// ─── Overview View ────────────────────────────────────────────────────────────
function renderOverview() {
  const openInc = state.incidents.filter(i => i.status === 'INVESTIGATING').length;
  const sev1 = state.incidents.filter(i => i.severity === 'SEV1' && i.status === 'INVESTIGATING').length;

  document.getElementById('global-stats').innerHTML = `
    ${statCard('Clusters', state.clusters.length, 'accent')}
    ${statCard('Open Incidents', openInc, openInc > 0 ? 'red' : 'green')}
    ${statCard('SEV1 Active', sev1, sev1 > 0 ? 'red' : 'green')}
    ${statCard('Avg MTTR', formatMTTR(state.stats.avg_mttr_secs), 'yellow')}
  `;

  document.getElementById('clusters-overview').innerHTML = state.clusters.length === 0
    ? emptyState('No clusters connected', 'Deploy the agent to a cluster and point it to this hub.')
    : state.clusters.map(c => clusterCard(c)).join('');

  const recent = state.incidents.slice(0, 10);
  document.getElementById('recent-incidents-table').innerHTML =
    incidentsTable(recent);
}

// ─── Clusters View ────────────────────────────────────────────────────────────
function renderClustersView() {
  document.getElementById('clusters-detail').innerHTML = state.clusters.length === 0
    ? emptyState('No clusters connected', 'Deploy the agent to a cluster and point it to this hub.')
    : state.clusters.map(c => clusterCard(c, true)).join('');
}

// ─── All Incidents View ───────────────────────────────────────────────────────
function renderAllIncidents(filtered) {
  const incidents = filtered || state.incidents;
  document.getElementById('all-incidents-table').innerHTML = incidentsTable(incidents);
}

function filterIncidents() {
  const cluster = document.getElementById('filter-cluster').value;
  const severity = document.getElementById('filter-severity').value;
  const status = document.getElementById('filter-status').value;

  let filtered = state.incidents;
  if (cluster) filtered = filtered.filter(i => i.cluster === cluster);
  if (severity) filtered = filtered.filter(i => i.severity === severity);
  if (status) filtered = filtered.filter(i => i.status === status);
  renderAllIncidents(filtered);
}

function populateClusterFilter() {
  const sel = document.getElementById('filter-cluster');
  const current = sel.value;
  sel.innerHTML = '<option value="">All Clusters</option>' +
    state.clusters.map(c => `<option value="${c.name}" ${c.name === current ? 'selected' : ''}>${c.name}</option>`).join('');
}

// ─── Analytics View ───────────────────────────────────────────────────────────
function renderAnalytics() {
  const resolved = state.incidents.filter(i => i.status === 'RESOLVED' || i.status === 'MITIGATED').length;
  document.getElementById('analytics-stats').innerHTML = `
    ${statCard('Total Incidents', state.stats.total_incidents || 0, 'accent')}
    ${statCard('Open', state.stats.open_incidents || 0, 'yellow')}
    ${statCard('Resolved / Mitigated', resolved, 'green')}
    ${statCard('Avg MTTR', formatMTTR(state.stats.avg_mttr_secs), 'yellow')}
  `;
}

// ─── Components ───────────────────────────────────────────────────────────────
function statCard(label, value, color = '') {
  return `
    <div class="stat-card">
      <div class="stat-label">${label}</div>
      <div class="stat-value ${color}">${value}</div>
    </div>`;
}

function clusterCard(c, expanded = false) {
  const score = Math.round(c.health_score || 0);
  const healthClass = score >= 80 ? 'health-good' : score >= 50 ? 'health-warn' : 'health-crit';
  const dot = score >= 80 ? '●' : score >= 50 ? '●' : '●';
  const ns = (c.namespaces || []).slice(0, 6);

  const clusterIncidents = state.incidents.filter(i => i.cluster === c.name && i.status === 'INVESTIGATING');
  const watchState = state.watchTimers[c.name];

  return `
    <div class="cluster-card">
      <div class="cluster-header">
        <div class="cluster-name">${escHtml(c.name)}</div>
        <div class="health-badge ${healthClass}">${dot} ${score}/100</div>
      </div>
      <div class="cluster-stats">
        <div class="cluster-stat">Nodes <span>${c.ready_nodes || 0}/${c.total_nodes || 0} Ready</span></div>
        <div class="cluster-stat">Pods Running <span>${c.running_pods || 0}</span></div>
        <div class="cluster-stat">Pending <span>${c.pending_pods || 0}</span></div>
        <div class="cluster-stat">CrashLoop <span style="color:var(--${c.crashloop > 0 ? 'red' : 'text'})">${c.crashloop || 0}</span></div>
      </div>
      ${ns.length > 0 ? `<div class="cluster-namespaces">${ns.map(n => `<span class="ns-badge">${escHtml(n)}</span>`).join('')}</div>` : ''}
      ${clusterIncidents.length > 0 ? `<div style="font-size:12px;color:var(--red);margin:8px 0">⚠ ${clusterIncidents.length} open incident${clusterIncidents.length > 1 ? 's' : ''}</div>` : ''}
      <div class="cluster-actions">
        <button class="btn btn-ghost btn-sm" onclick="triggerScan('${escHtml(c.name)}')">🔍 Scan Once</button>
        ${watchState && watchState.running
          ? `<button class="btn btn-danger btn-sm" onclick="stopWatch('${escHtml(c.name)}')">⏸ Stop Watching</button>`
          : `<button class="btn btn-accent btn-sm" onclick="startWatch('${escHtml(c.name)}')">▶ Watch</button>`
        }
      </div>
      ${watchState && watchState.running ? `<div class="watch-info">Next scan in ${formatCountdown(watchState.remaining)}</div>` : ''}
    </div>`;
}

function incidentsTable(incidents) {
  if (!incidents || incidents.length === 0) {
    return emptyState('No incidents found', 'Your cluster is healthy or no scans have been run yet.');
  }
  return `
    <table class="incidents-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Severity</th>
          <th>Title</th>
          <th>Cluster</th>
          <th>Namespace</th>
          <th>Status</th>
          <th>Confidence</th>
          <th>Detected</th>
        </tr>
      </thead>
      <tbody>
        ${incidents.map(i => `
          <tr onclick="openIncident('${i.id}')">
            <td style="font-family:var(--mono);font-size:12px;color:var(--text2)">${i.id}</td>
            <td>${sevBadge(i.severity, i.status === 'INVESTIGATING')}</td>
            <td style="max-width:280px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${escHtml(i.title || '')}</td>
            <td style="font-family:var(--mono);font-size:12px">${escHtml(i.cluster)}</td>
            <td style="font-family:var(--mono);font-size:12px">${escHtml(i.namespace || 'all')}</td>
            <td>${statusBadge(i.status)}</td>
            <td style="color:var(--text2)">${i.confidence_score || 0}%</td>
            <td style="color:var(--text2);font-size:12px;white-space:nowrap">${formatDate(i.detected_at)}</td>
          </tr>`).join('')}
      </tbody>
    </table>`;
}

// ─── Incident Modal ───────────────────────────────────────────────────────────
async function openIncident(id) {
  const overlay = document.getElementById('modal-overlay');
  const content = document.getElementById('modal-content');
  overlay.classList.remove('hidden');
  content.innerHTML = `<div style="text-align:center;padding:40px"><span class="spinner"></span>Loading…</div>`;

  try {
    const inc = await fetch(`${API}/api/v1/incidents/${id}`).then(r => r.json());
    content.innerHTML = renderIncidentDetail(inc);
  } catch (e) {
    content.innerHTML = `<div style="color:var(--red)">Failed to load incident</div>`;
  }
}

function renderIncidentDetail(inc) {
  const affected = tryParseJSON(inc.affected_services) || [];
  const factors = tryParseJSON(inc.contributing_factors) || [];
  const plan = tryParseJSON(inc.remediation_plan) || [];
  const namespaces = tryParseJSON(inc.affected_namespaces) || [];

  return `
    <div>
      <div style="display:flex;align-items:center;gap:10px;margin-bottom:6px">
        ${sevBadge(inc.severity)}
        ${statusBadge(inc.status)}
      </div>
      <h2 class="incident-detail-title">${escHtml(inc.title || 'Untitled Incident')}</h2>
      <div class="incident-meta">
        <div class="incident-meta-item">ID <span>${inc.id}</span></div>
        <div class="incident-meta-item">Cluster <span>${inc.cluster}</span></div>
        <div class="incident-meta-item">Namespaces <span>${namespaces.join(', ') || inc.namespace || '—'}</span></div>
        <div class="incident-meta-item">Category <span>${inc.category || '—'}</span></div>
        <div class="incident-meta-item">Confidence <span>${inc.confidence_score || 0}%</span></div>
        <div class="incident-meta-item">Detected <span>${formatDate(inc.detected_at)}</span></div>
        <div class="incident-meta-item">LLM <span>${inc.llm_model || '—'}</span></div>
      </div>

      ${affected.length > 0 ? `
        <div class="section-block">
          <h3>Affected Pods / Services</h3>
          <div class="affected-pods">
            ${affected.map(s => `<span class="pod-chip ${s.impact_level}">${escHtml(s.service_name)} <span style="opacity:0.6">${s.namespace}</span> — ${s.impact_level}</span>`).join('')}
          </div>
        </div>` : ''}

      ${inc.root_cause ? `
        <div class="section-block">
          <h3>Root Cause</h3>
          <p>${escHtml(inc.root_cause)}</p>
        </div>` : ''}

      ${factors.length > 0 ? `
        <div class="section-block">
          <h3>Contributing Factors</h3>
          <ul class="factor-list">${factors.map(f => `<li>${escHtml(f)}</li>`).join('')}</ul>
        </div>` : ''}

      ${plan.length > 0 ? `
        <div class="section-block">
          <h3>Remediation Plan</h3>
          <div class="remediation-steps">
            ${plan.map(step => `
              <div class="remediation-step">
                <div class="step-header">
                  <span class="step-num">STEP ${step.step_number}</span>
                  ${step.is_destructive ? '<span class="destructive-tag">⚠ DESTRUCTIVE</span>' : ''}
                </div>
                <div class="step-desc">${escHtml(step.description)}</div>
                ${step.command ? `
                  <div class="step-cmd">
                    <code>${escHtml(step.command)}</code>
                    <button class="copy-btn" onclick="copyCmd('${escHtml(step.command)}')" title="Copy command">⎘</button>
                  </div>` : ''}
              </div>`).join('')}
          </div>
        </div>` : ''}

      <div style="display:flex;gap:10px;margin-top:20px;padding-top:16px;border-top:1px solid var(--border)">
        <button class="btn btn-ghost btn-sm" onclick="updateStatus('${inc.id}', 'MITIGATED')">Mark Mitigated</button>
        <button class="btn btn-ghost btn-sm" onclick="updateStatus('${inc.id}', 'RESOLVED')">Mark Resolved</button>
      </div>
    </div>`;
}

function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden');
}

// ─── Cluster Actions ──────────────────────────────────────────────────────────
async function triggerScan(clusterName) {
  showToast(`Triggering scan for ${clusterName}…`);
  try {
    // The hub proxies to the agent
    const r = await fetch(`${API}/api/v1/clusters/${encodeURIComponent(clusterName)}/trigger`, { method: 'POST' });
    if (r.ok) {
      showToast(`✓ Scan started for ${clusterName}`, 'green');
      setTimeout(refreshAll, 5000);
    } else {
      showToast('Scan trigger failed — check agent connectivity', 'red');
    }
  } catch (e) {
    showToast('Cannot reach hub API', 'red');
  }
}

async function startWatch(clusterName) {
  try {
    await fetch(`${API}/api/v1/clusters/${encodeURIComponent(clusterName)}/watch`, { method: 'POST' });
    state.watchTimers[clusterName] = { running: true, remaining: 300 };
    startCountdown(clusterName);
    renderCurrentView();
    showToast(`▶ Watching ${clusterName}`, 'green');
  } catch (e) {
    showToast('Failed to start watch', 'red');
  }
}

async function stopWatch(clusterName) {
  try {
    await fetch(`${API}/api/v1/clusters/${encodeURIComponent(clusterName)}/watch`, { method: 'DELETE' });
    clearInterval(state.watchTimers[clusterName]?.interval);
    delete state.watchTimers[clusterName];
    renderCurrentView();
    showToast(`⏸ Watch stopped for ${clusterName}`, '');
  } catch (e) {
    showToast('Failed to stop watch', 'red');
  }
}

function startCountdown(clusterName) {
  const t = state.watchTimers[clusterName];
  t.interval = setInterval(() => {
    if (!state.watchTimers[clusterName]) return;
    state.watchTimers[clusterName].remaining--;
    if (state.watchTimers[clusterName].remaining <= 0) {
      state.watchTimers[clusterName].remaining = 300;
      refreshAll();
    }
    renderCurrentView();
  }, 1000);
}

async function updateStatus(id, status) {
  await fetch(`${API}/api/v1/incidents/${id}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });
  closeModal();
  await refreshAll();
  showToast(`Incident marked as ${status}`, 'green');
}

// ─── Helpers ──────────────────────────────────────────────────────────────────
function copyCmd(cmd) {
  navigator.clipboard.writeText(cmd).then(() => showToast('Command copied!', 'green'));
}

function sevBadge(sev, pulse = false) {
  const cls = (sev || 'SEV4').toLowerCase();
  return `<span class="sev ${cls} ${pulse && cls === 'sev1' ? 'sev1-pulse' : ''}">${sev || 'SEV4'}</span>`;
}

function statusBadge(status) {
  const map = { INVESTIGATING: 'status-investigating', MITIGATED: 'status-mitigated', RESOLVED: 'status-resolved' };
  return `<span class="status-badge ${map[status] || ''}">${status || 'UNKNOWN'}</span>`;
}

function emptyState(title, subtitle) {
  return `<div class="empty-state"><div class="big-icon">◈</div><p><strong>${title}</strong></p><p>${subtitle}</p></div>`;
}

function formatDate(d) {
  if (!d) return '—';
  return new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function formatMTTR(secs) {
  if (!secs) return '—';
  const m = Math.round(secs / 60);
  return m < 60 ? `${m}m` : `${Math.round(m / 60)}h`;
}

function formatCountdown(s) {
  const m = Math.floor(s / 60), sec = s % 60;
  return `${m}m ${sec < 10 ? '0' : ''}${sec}s`;
}

function escHtml(str) {
  const div = document.createElement('div');
  div.appendChild(document.createTextNode(String(str || '')));
  return div.innerHTML;
}

function tryParseJSON(val) {
  if (!val) return null;
  if (typeof val === 'object') return val;
  try { return JSON.parse(val); } catch { return null; }
}

// ─── Toast ────────────────────────────────────────────────────────────────────
let toastTimeout;
function showToast(msg, color = '') {
  let toast = document.getElementById('toast');
  if (!toast) {
    toast = document.createElement('div');
    toast.id = 'toast';
    toast.style.cssText = `
      position:fixed;bottom:24px;right:24px;background:var(--bg3);border:1px solid var(--border);
      border-radius:8px;padding:12px 20px;font-size:13px;z-index:9999;
      box-shadow:0 8px 32px rgba(0,0,0,0.5);transition:opacity 0.3s;`;
    document.body.appendChild(toast);
  }
  toast.style.color = color === 'green' ? 'var(--green)' : color === 'red' ? 'var(--red)' : 'var(--text)';
  toast.textContent = msg;
  toast.style.opacity = '1';
  clearTimeout(toastTimeout);
  toastTimeout = setTimeout(() => { toast.style.opacity = '0'; }, 3000);
}

// ─── Boot ─────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  fetchAll();
  setInterval(fetchAll, POLL_INTERVAL);
});
