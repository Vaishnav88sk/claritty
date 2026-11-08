<script>
  let { label = '', value = '', color = '' } = $props();
</script>

<div class="stat-card">
  <div class="stat-label">{label}</div>
  <div class="stat-value {color}">{value}</div>
</div>

