// Utility functions for Claritty SRE Hub Dashboard

export function formatDate(d) {
  if (!d) return '—';
  return new Date(d).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
}

export function formatTimeOnly(d) {
  if (!d) return '—';
  try {
    return new Date(d).toLocaleTimeString(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
  } catch {
    return String(d);
  }
}

export function formatMTTR(secs) {
  if (!secs) return '—';
  const m = Math.round(secs / 60);
  return m < 60 ? `${m}m` : `${Math.round(m / 60)}h`;
}

export function formatCountdown(s) {
  const m = Math.floor(s / 60);
  const sec = s % 60;
  return `${m}m ${sec < 10 ? '0' : ''}${sec}s`;
}

export function tryParseJSON(val) {
  if (!val) return null;
  if (typeof val === 'object') return val;
  try {
    return JSON.parse(val);
  } catch {
    return null;
  }
}

export function sevClass(sev) {
  return (sev || 'sev4').toLowerCase();
}

export function statusClass(status) {
  const map = {
    INVESTIGATING: 'status-investigating',
    MITIGATED: 'status-mitigated',
    RESOLVED: 'status-resolved'
  };
  return map[status] || '';
}

