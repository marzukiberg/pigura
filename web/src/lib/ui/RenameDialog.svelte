<script>
  import { tick } from 'svelte';

  // Dialog ganti nama: satu kolom teks. onsubmit(nama) boleh async; lempar Error untuk menampilkan pesan.
  let {
    open = false,
    title = 'Ganti nama',
    value = '',
    maxlength = 200,
    submitLabel = 'Simpan',
    onsubmit,
    oncancel,
  } = $props();

  let text = $state('');
  let busy = $state(false);
  let error = $state('');
  let input = $state();

  // Setiap dibuka: isi ulang dengan nama saat ini, fokus, dan pilih seluruh teks.
  $effect(() => {
    if (!open) return;
    text = value;
    error = '';
    busy = false;
    tick().then(() => {
      input?.focus();
      input?.select();
    });
  });

  let trimmed = $derived(text.trim());
  let unchanged = $derived(trimmed === value.trim());

  async function submit(e) {
    e.preventDefault();
    if (!trimmed || busy) return;
    if (unchanged) return oncancel?.();
    busy = true;
    error = '';
    try {
      await onsubmit?.(trimmed);
    } catch (err) {
      error = err?.message || 'Gagal menyimpan.';
      busy = false;
    }
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && open && !busy && oncancel?.()} />

{#if open}
  <div
    class="backdrop"
    role="presentation"
    onclick={(e) => e.target === e.currentTarget && !busy && oncancel?.()}
  >
    <div class="card" role="dialog" aria-modal="true" aria-labelledby="rn-title">
      <form onsubmit={submit}>
      <h3 id="rn-title">{title}</h3>
      <input
        bind:this={input}
        bind:value={text}
        {maxlength}
        autocomplete="off"
        aria-label={title}
        aria-invalid={!!error}
        disabled={busy}
      />
      {#if error}<p class="err" role="alert">{error}</p>{/if}
      <div class="actions">
        <button type="button" class="cancel" onclick={() => oncancel?.()} disabled={busy}>Batal</button>
        <button class="ok" disabled={!trimmed || busy}>{busy ? 'Menyimpan…' : submitLabel}</button>
      </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .backdrop {
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
  }
  .card {
    background: var(--surface, #fff);
    color: var(--fg, #202124);
    border: 1px solid var(--line, #e0e0e0);
    border-radius: 24px;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
    width: 100%;
    max-width: 380px;
    padding: 22px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  form { display: flex; flex-direction: column; gap: 14px; }
  h3 { margin: 0; font-size: 18px; font-weight: 600; }
  input {
    height: 44px;
    padding: 0 14px;
    border-radius: 12px;
    border: 1px solid var(--line);
    background: var(--bg);
    color: var(--fg);
    font: inherit;
    font-size: 15px;
    width: 100%;
    box-sizing: border-box;
  }
  input:focus { outline: 2px solid var(--acc); border-color: transparent; }
  .err { margin: 0; font-size: 13px; color: #d93025; }
  .actions { display: flex; gap: 10px; justify-content: flex-end; }
  .cancel, .ok { height: 40px; padding: 0 20px; border-radius: 20px; font-size: 14px; font-weight: 500; }
  .cancel { background: var(--surface-variant); color: var(--fg); }
  .cancel:hover:not(:disabled) { background: var(--hover); }
  .ok { background: var(--acc-bg); color: var(--acc-fg); }
  .ok:hover:not(:disabled) { filter: brightness(0.96); }
  .cancel:disabled, .ok:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
