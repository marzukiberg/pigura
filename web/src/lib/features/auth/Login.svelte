<script>
  import Icon from '../../ui/Icon.svelte';
  import { login } from '../../core/auth.svelte.js';

  let token = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(e) {
    e.preventDefault();
    if (!token || busy) return;
    busy = true;
    error = (await login(token)) ?? '';
    busy = false;
    if (!error) location.reload();
  }
</script>

<main class="shell">
  <form class="card" onsubmit={submit}>
    <div class="mark"><Icon name="storage" size={26} /></div>
    <h1>Masukkan kode akses</h1>
    <p>Server ini dilindungi. Kode disimpan sebagai sesi di perangkat ini.</p>
    <input
      type="password"
      bind:value={token}
      autocomplete="current-password"
      placeholder="Kode akses"
      aria-label="Kode akses"
      aria-invalid={!!error}
    />
    {#if error}<p class="err" role="alert">{error}</p>{/if}
    <button class="go" disabled={!token || busy}>{busy ? 'Memeriksa…' : 'Masuk'}</button>
  </form>
</main>

<style>
  .shell { min-height: 100dvh; display: grid; place-items: center; padding: 24px; background: var(--bg); }
  .card {
    width: min(100%, 400px); padding: 28px; border-radius: 24px; background: var(--surface);
    border: 1px solid var(--line); box-shadow: var(--card-shadow); display: flex; flex-direction: column; gap: 14px;
  }
  .mark { width: 48px; height: 48px; border-radius: 14px; display: grid; place-items: center; background: var(--acc-bg); color: var(--acc-fg); }
  h1 { font-size: 22px; font-weight: 500; color: var(--fg); }
  p { margin: 0; font-size: 13px; color: var(--mut); line-height: 1.45; }
  input { height: 46px; padding: 0 14px; border-radius: 12px; border: 1px solid var(--line); background: var(--bg); color: var(--fg); font: inherit; }
  input:focus { outline: 2px solid var(--acc); border-color: transparent; }
  input[aria-invalid='true'] { border-color: #d93025; }
  .err { color: #d93025; }
  .go { height: 44px; border-radius: 22px; background: var(--acc-bg); color: var(--acc-fg); font-weight: 500; font-size: 15px; }
  .go:disabled { opacity: 0.5; }
</style>
