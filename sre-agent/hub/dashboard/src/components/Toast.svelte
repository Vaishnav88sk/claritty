<script>
  let { message = '', color = '', visible = false } = $props();

  let toastColor = $derived(
    color === 'green'
      ? 'var(--green)'
      : color === 'red'
      ? 'var(--red)'
      : 'var(--text)'
  );
</script>

{#if visible}
  <div
    id="toast"
    style="
      position: fixed;
      bottom: 24px;
      right: 24px;
      background: var(--bg3);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px 20px;
      font-size: 13px;
      z-index: 9999;
      box-shadow: 0 8px 32px rgba(0,0,0,0.5);
      transition: opacity 0.3s;
      opacity: {visible ? 1 : 0};
      color: {toastColor};
    "
  >
    {message}
  </div>
{/if}

