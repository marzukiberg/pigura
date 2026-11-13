<script>
  import Icon from '../../ui/Icon.svelte';
  import { connection } from '../../core/api.svelte.js';
  import { setup, saveSetup } from '../../core/connection.svelte.js';
</script>

<main class="setup-shell">
  <section class="setup-card" aria-labelledby="setup-title">
    <div class="setup-mark"><Icon name="storage" size={26} /></div>
    <p class="setup-eyebrow">Pengaturan awal</p>
    {#if setup.auto}
      <h1 id="setup-title">Pilih folder target scan</h1>
      <p class="setup-copy">Server belum punya folder library. Pilih folder yang berisi foto dan video untuk dipindai.</p>
    {:else}
      <h1 id="setup-title">Hubungkan server dan pilih target scan</h1>
      <p class="setup-copy">Aplikasi tidak akan memuat galeri sebelum kedua pengaturan ini tersimpan.</p>
      <label>
        <span>URL proxy / server</span>
        <input bind:value={connection.baseUrl} inputmode="url" placeholder="http://100.x.y.z:PORT" />
        <small>Alamat server Pigura (mis. http://192.168.1.10:8080). UI yang dilayani server itu sendiri tidak perlu mengisi ini.</small>
      </label>
    {/if}
    <label>
      <span>Folder target scan</span>
      <input bind:value={connection.library} placeholder={setup.scanBase ? `${setup.scanBase}/…` : '/srv/photos'} />
      <small>Harus berada di bawah {setup.scanBase || 'base scan'} (folder induk yang diizinkan server).</small>
    </label>
    {#if setup.error}<p class="setup-error" role="alert">{setup.error}</p>{/if}
    <button class="setup-submit" onclick={saveSetup} disabled={setup.saving}>
      {setup.saving ? 'Menyimpan…' : 'Simpan dan lanjutkan'}
    </button>
  </section>
</main>

<style>
  .setup-shell {
    min-height: 100dvh;
    display: grid;
    place-items: center;
    padding: 24px;
    background: var(--bg);
  }

  .setup-card {
    width: min(100%, 480px);
    padding: 32px;
    border: 1px solid var(--line-subtle);
    border-radius: 24px;
    background: var(--surface);
    box-shadow: var(--card-shadow);
  }

  .setup-mark { color: var(--acc); margin-bottom: 20px; }

  .setup-eyebrow { margin: 0 0 8px; color: var(--acc-fg); font-size: 13px; font-weight: 700; text-transform: uppercase; letter-spacing: .08em; }

  .setup-card h1 { margin: 0; font-size: 26px; letter-spacing: -.6px; }

  .setup-copy { color: var(--mut); line-height: 1.5; margin: 12px 0 24px; }

  .setup-card label { display: grid; gap: 8px; margin-top: 16px; color: var(--fg); font-size: 14px; font-weight: 600; }

  .setup-card input { width: 100%; box-sizing: border-box; border: 1px solid var(--line); border-radius: 10px; padding: 12px; background: var(--bg); color: var(--fg); font: inherit; font-weight: 400; }

  .setup-card small { color: var(--mut); font-size: 12px; font-weight: 400; line-height: 1.4; }

  .setup-error { color: #b3261e; font-size: 13px; margin: 16px 0 0; }

  .setup-submit { width: 100%; margin-top: 24px; padding: 13px 16px; border-radius: 12px; background: var(--acc); color: white; font: inherit; font-weight: 700; }

  .setup-submit:disabled { opacity: .65; }
</style>
