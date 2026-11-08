<script>
  let { title = '', subtitle = '' } = $props();
</script>

<div class="empty-state">
  <div class="big-icon">◈</div>
  <p><strong>{title}</strong></p>
  <p>{subtitle}</p>
</div>

