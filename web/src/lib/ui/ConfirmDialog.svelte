<script>
  import Icon from './Icon.svelte';

  let {
    open = false,
    title = 'Hapus foto?',
    message = 'Foto akan dihapus dari library.',
    confirmLabel = 'Hapus',
    cancelLabel = 'Batal',
    loading = false,
    onconfirm,
    oncancel,
  } = $props();

  function onKeydown(e) {
    if (e.key === 'Escape' && open && !loading) {
      oncancel?.();
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <div
    class="dialog-backdrop"
    role="presentation"
    onclick={(e) => {
      if (e.target === e.currentTarget && !loading) oncancel?.();
    }}
  >
    <div
      class="dialog-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
      aria-describedby="dialog-desc"
    >
      <div class="dialog-icon-wrap">
        <Icon name="trash" size={24} />
      </div>

      <div class="dialog-content">
        <h3 id="dialog-title">{title}</h3>
        <p id="dialog-desc">{message}</p>
      </div>

      <div class="dialog-actions">
        <button class="btn-cancel" onclick={() => oncancel?.()} disabled={loading}>
          {cancelLabel}
        </button>
        <button class="btn-delete" onclick={() => onconfirm?.()} disabled={loading}>
          {#if loading}
            <span class="btn-spinner"></span>
            <span>Menghapus…</span>
          {:else}
            <Icon name="trash" size={16} />
            <span>{confirmLabel}</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .dialog-backdrop {
    position: fixed;
    inset: 0;
    z-index: 200;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    animation: fadeIn 0.15s ease-out;
  }

  .dialog-card {
    background: var(--surface, #ffffff);
    color: var(--fg, #202124);
    border: 1px solid var(--line-subtle, #e0e0e0);
    border-radius: 28px;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
    width: 100%;
    max-width: 360px;
    padding: 24px;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    animation: scaleIn 0.18s cubic-bezier(0.1, 0.9, 0.2, 1);
  }

  @media (prefers-color-scheme: dark) {
    .dialog-card {
      background: #28292c;
      color: #e8eaed;
      border-color: rgba(255, 255, 255, 0.08);
      box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
    }
  }

  .dialog-icon-wrap {
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: rgba(234, 67, 53, 0.12);
    color: #ea4335;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 16px;
  }

  .dialog-content h3 {
    font-size: 19px;
    font-weight: 600;
    margin: 0 0 8px;
    color: var(--fg, #202124);
  }

  @media (prefers-color-scheme: dark) {
    .dialog-content h3 {
      color: #f1f3f4;
    }
  }

  .dialog-content p {
    font-size: 14px;
    line-height: 1.5;
    color: var(--mut, #5f6368);
    margin: 0 0 24px;
  }

  .dialog-actions {
    display: flex;
    gap: 10px;
    width: 100%;
    justify-content: flex-end;
  }

  .btn-cancel {
    flex: 1;
    height: 42px;
    border-radius: 21px;
    background: var(--surface-variant, rgba(0, 0, 0, 0.05));
    color: var(--fg, #202124);
    font-size: 14px;
    font-weight: 500;
    transition: background 0.15s;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  @media (prefers-color-scheme: dark) {
    .btn-cancel {
      background: rgba(255, 255, 255, 0.08);
      color: #e8eaed;
    }
  }

  .btn-cancel:hover:not(:disabled) {
    background: var(--hover, rgba(0, 0, 0, 0.1));
  }

  .btn-delete {
    flex: 1;
    height: 42px;
    border-radius: 21px;
    background: #d93025;
    color: #ffffff;
    font-size: 14px;
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: background 0.15s, transform 0.1s;
  }

  .btn-delete:hover:not(:disabled) {
    background: #c5221f;
  }

  .btn-delete:active:not(:disabled) {
    transform: scale(0.98);
  }

  .btn-delete:disabled,
  .btn-cancel:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-spinner {
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: #ffffff;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes scaleIn {
    from {
      opacity: 0;
      transform: scale(0.92);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
