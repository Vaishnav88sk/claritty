// API client for Claritty SRE Hub

const API = '';

export async function fetchClusters() {
  const res = await fetch(`${API}/api/v1/clusters`);
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res.json();
}

export async function fetchIncidents(limit = 200) {
  const res = await fetch(`${API}/api/v1/incidents?limit=${limit}`);
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res.json();
}

export async function fetchStats() {
  const res = await fetch(`${API}/api/v1/stats`);
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res.json();
}

export async function fetchIncident(id) {
  const res = await fetch(`${API}/api/v1/incidents/${id}`);
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res.json();
}

export async function triggerScan(clusterName) {
  const res = await fetch(`${API}/api/v1/clusters/${encodeURIComponent(clusterName)}/trigger`, {
    method: 'POST'
  });
  return res;
}

export async function startWatch(clusterName) {
  const res = await fetch(`${API}/api/v1/clusters/${encodeURIComponent(clusterName)}/watch`, {
    method: 'POST'
  });
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res;
}

export async function stopWatch(clusterName) {
  const res = await fetch(`${API}/api/v1/clusters/${encodeURIComponent(clusterName)}/watch`, {
    method: 'DELETE'
  });
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res;
}

export async function updateIncidentStatus(id, status) {
  const res = await fetch(`${API}/api/v1/incidents/${id}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status })
  });
  if (!res.ok) throw new Error(`HTTP error ${res.status}`);
  return res;
}

