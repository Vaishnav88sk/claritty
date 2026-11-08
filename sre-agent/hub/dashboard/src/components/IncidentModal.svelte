<script>
  import {
    formatDate,
    formatTimeOnly,
    tryParseJSON,
    sevClass,
    statusClass
  } from '../lib/utils.js';

  let {
    incident = null,
    loading = false,
    error = null,
    onClose = () => {},
    onUpdateStatus = () => {},
    onCopyCmd = () => {}
  } = $props();

  let affected = $derived(incident ? tryParseJSON(incident.affected_services) || [] : []);
  let factors = $derived(incident ? tryParseJSON(incident.contributing_factors) || [] : []);
  let plan = $derived(incident ? tryParseJSON(incident.remediation_plan) || [] : []);
  let namespaces = $derived(incident ? tryParseJSON(incident.affected_namespaces) || [] : []);
  let timeline = $derived(incident ? tryParseJSON(incident.timeline) || [] : []);
  let changes = $derived(incident ? tryParseJSON(incident.changes) || [] : []);
  let conclusions = $derived(incident ? tryParseJSON(incident.conclusions) || [] : []);
  let evidence = $derived(incident ? tryParseJSON(incident.evidence) || [] : []);
  let hypotheses = $derived(incident ? tryParseJSON(incident.hypotheses) || [] : []);
</script>

<div
  class="modal-overlay"
  id="modal-overlay"
  role="presentation"
  onclick={onClose}
>
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="modal"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => e.stopPropagation()}
  >
    <button class="modal-close" onclick={onClose} aria-label="Close modal">✕</button>
    <div id="modal-content">
      {#if loading}
        <div style="text-align:center;padding:40px">
          <span class="spinner"></span>Loading…
        </div>
      {:else if error}
        <div style="color:var(--red);padding:20px;text-align:center">{error}</div>
      {:else if incident}
        <div>
          <div style="display:flex;align-items:center;gap:10px;margin-bottom:6px">
            <span class="sev {sevClass(incident.severity)}">
              {incident.severity || 'SEV4'}
            </span>
            <span class="status-badge {statusClass(incident.status)}">
              {incident.status || 'UNKNOWN'}
            </span>
          </div>
          <h2 class="incident-detail-title">
            {incident.title || 'Untitled Incident'}
          </h2>
          <div class="incident-meta">
            <div class="incident-meta-item">ID <span>{incident.id}</span></div>
            <div class="incident-meta-item">Cluster <span>{incident.cluster}</span></div>
            <div class="incident-meta-item">
              Namespaces <span>{namespaces.join(', ') || incident.namespace || '—'}</span>
            </div>
            <div class="incident-meta-item">
              Category <span>{incident.category || '—'}</span>
            </div>
            <div class="incident-meta-item">
              Confidence <span>{incident.confidence_score || 0}%</span>
            </div>
            <div class="incident-meta-item">
              Detected <span>{formatDate(incident.detected_at)}</span>
            </div>
            <div class="incident-meta-item">
              LLM <span>{incident.llm_model || '—'}</span>
            </div>
          </div>

          {#if affected.length > 0}
            <div class="section-block">
              <h3>Affected Pods / Services</h3>
              <div class="affected-pods">
                {#each affected as s}
                  <span class="pod-chip {s.impact_level}">
                    {s.service_name} <span style="opacity:0.6">{s.namespace}</span> — {s.impact_level}
                  </span>
                {/each}
              </div>
            </div>
          {/if}

          {#if incident.root_cause}
            <div class="section-block">
              <h3>Root Cause</h3>
              <p>{incident.root_cause}</p>
            </div>
          {/if}

          {#if timeline.length > 0}
            <div class="section-block">
              <h3>Incident Timeline</h3>
              <div class="timeline-container">
                {#each timeline as e}
                  <div class="timeline-row">
                    <span class="timeline-time">{formatTimeOnly(e.timestamp)}</span>
                    <span class="timeline-badge {e.severity || 'info'}">
                      {e.source || 'k8s'}
                    </span>
                    <div class="timeline-body">
                      {#if e.resource}
                        <span class="timeline-resource">[{e.resource}]</span>{' '}
                      {/if}
                      <span class="timeline-msg">{e.message}</span>
                      {#if e.evidence_id}
                        <span class="evidence-tag">[{e.evidence_id}]</span>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if changes.length > 0}
            <div class="section-block">
              <h3>Recent Changes Investigated ("What Changed?")</h3>
              <div class="changes-container">
                {#each changes as c}
                  <div class="change-row">
                    <span class="timeline-time">{formatTimeOnly(c.timestamp)}</span>
                    <span class="change-cat">{c.category}</span>
                    <div class="change-body">
                      <span class="timeline-resource">{c.resource}:</span>
                      <span>{c.summary}</span>
                      {#if c.correlated}
                        <span class="correlated-badge">CORRELATED</span>
                      {/if}
                      {#if c.evidence_id}
                        <span class="evidence-tag">[{c.evidence_id}]</span>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if conclusions.length > 0}
            <div class="section-block">
              <h3>Evidence-Backed Conclusions</h3>
              <div class="conclusions-list">
                {#each conclusions as c}
                  <div class="conclusion-card">
                    <div class="conclusion-stmt">
                      <strong>• {c.statement}</strong>
                      {#if c.evidence_ids && c.evidence_ids.length > 0}
                        <span class="evidence-tag">[{c.evidence_ids.join(', ')}]</span>
                      {/if}
                    </div>
                    {#if c.reasoning}
                      <div class="conclusion-reason">Reasoning: {c.reasoning}</div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if evidence.length > 0}
            <div class="section-block">
              <h3>Supporting Evidence</h3>
              <div class="evidence-list">
                {#each evidence as ev}
                  <div class="evidence-row">
                    <span class="evidence-id">{ev.id}</span>
                    <span class="evidence-sev {ev.severity}">{ev.severity}</span>
                    <div class="evidence-content">
                      {#if ev.resource}
                        <span class="timeline-resource">[{ev.resource}]</span>{' '}
                      {/if}
                      <span>{ev.content}</span>
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if hypotheses.length > 0}
            <div class="section-block">
              <h3 style="color:var(--yellow)">Hypotheses (Unverified / Requires Validation)</h3>
              <div class="hypotheses-list">
                {#each hypotheses as h}
                  <div class="hypothesis-card">
                    <div class="hypothesis-stmt"><strong>• {h.statement}</strong></div>
                    {#if h.reason}
                      <div class="hypothesis-reason">Why: {h.reason}</div>
                    {/if}
                    {#if h.validation_steps && h.validation_steps.length > 0}
                      <div class="hypothesis-steps">
                        Validation: {h.validation_steps.join('; ')}
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          {#if factors.length > 0}
            <div class="section-block">
              <h3>Contributing Factors</h3>
              <ul class="factor-list">
                {#each factors as f}
                  <li>{f}</li>
                {/each}
              </ul>
            </div>
          {/if}

          {#if plan.length > 0}
            <div class="section-block">
              <h3>Remediation Plan</h3>
              <div class="remediation-steps">
                {#each plan as step}
                  <div class="remediation-step">
                    <div class="step-header">
                      <span class="step-num">STEP {step.step_number}</span>
                      {#if step.is_destructive}
                        <span class="destructive-tag">⚠ DESTRUCTIVE</span>
                      {/if}
                    </div>
                    <div class="step-desc">{step.description}</div>
                    {#if step.command}
                      <div class="step-cmd">
                        <code>{step.command}</code>
                        <button
                          class="copy-btn"
                          onclick={() => onCopyCmd(step.command)}
                          title="Copy command"
                        >
                          ⎘
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          <div
            style="display:flex;gap:10px;margin-top:20px;padding-top:16px;border-top:1px solid var(--border)"
          >
            <button
              class="btn btn-ghost btn-sm"
              onclick={() => onUpdateStatus(incident.id, 'MITIGATED')}
            >
              Mark Mitigated
            </button>
            <button
              class="btn btn-ghost btn-sm"
              onclick={() => onUpdateStatus(incident.id, 'RESOLVED')}
            >
              Mark Resolved
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
