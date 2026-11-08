<script>
  import { formatDate, sevClass, statusClass } from '../lib/utils.js';
  import EmptyState from './EmptyState.svelte';

  let { incidents = [], onSelect = () => {} } = $props();
</script>

{#if !incidents || incidents.length === 0}
  <EmptyState
    title="No incidents found"
    subtitle="Your cluster is healthy or no scans have been run yet."
  />
{:else}
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
      {#each incidents as incident (incident.id)}
        {@const cls = sevClass(incident.severity)}
        {@const isInvestigating = incident.status === 'INVESTIGATING'}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <tr onclick={() => onSelect(incident.id)} style="cursor: pointer">
          <td style="font-family:var(--mono);font-size:12px;color:var(--text2)">
            {incident.id}
          </td>
          <td>
            <span class="sev {cls} {isInvestigating && cls === 'sev1' ? 'sev1-pulse' : ''}">
              {incident.severity || 'SEV4'}
            </span>
          </td>
          <td style="max-width:280px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
            {incident.title || ''}
          </td>
          <td style="font-family:var(--mono);font-size:12px">
            {incident.cluster}
          </td>
          <td style="font-family:var(--mono);font-size:12px">
            {incident.namespace || 'all'}
          </td>
          <td>
            <span class="status-badge {statusClass(incident.status)}">
              {incident.status || 'UNKNOWN'}
            </span>
          </td>
          <td style="color:var(--text2)">
            {incident.confidence_score || 0}%
          </td>
          <td style="color:var(--text2);font-size:12px;white-space:nowrap">
            {formatDate(incident.detected_at)}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
