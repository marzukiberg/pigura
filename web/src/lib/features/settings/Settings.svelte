<script>
  import Icon from '../../ui/Icon.svelte';
  import SharesList from '../share/SharesList.svelte';
  import Seg from '../../ui/Seg.svelte';
  import About from './About.svelte';
  import { connection } from '../../core/api.svelte.js';
  import { setup } from '../../core/connection.svelte.js';

  let { health, theme, scanning, saving, error, saved, onscan, onsave, ontheme } = $props();

  const THEMES = [
    { id: 'system', label: 'Sistem' },
    { id: 'light', label: 'Terang' },
    { id: 'dark', label: 'Gelap' },
  ];
</script>

<div class="settings">
  <h1>Pengaturan</h1>

  <div class="cols">
  <div class="col">
  <section>
    <h2>Koneksi</h2>
    {#if !setup.auto}
    <label>
      <span>URL proxy / server</span>
      <input bind:value={connection.baseUrl} inputmode="url" placeholder="http://127.0.0.1:5173" />
    </label>
    {/if}
    <label>
      <span>Folder target scan</span>
      <input bind:value={connection.library} placeholder="/srv/photos" />
      <small>Harus berada di bawah base scan yang diizinkan server.</small>
    </label>
    {#if error}<p class="msg err" role="alert">{error}</p>{/if}
    {#if saved && !error}<p class="msg ok" role="status">Tersimpan.</p>{/if}
    <button class="primary" onclick={onsave} disabled={saving}>{saving ? 'Menyimpan…' : 'Simpan'}</button>
  </section>

  <section>
    <h2>Tampilan</h2>
    <Seg role="tab" variant="inline" size="sm" px={18} label="Tema" options={THEMES} value={theme} onchange={ontheme} />
  </section>

  <section>
    <h2>Pustaka</h2>
    <div class="stat">
      <span class="ico"><Icon name="storage" size={18} /></span>
      <div>
        <div class="v">{health.assets} aset terindeks</div>
        <small>{health.scanning ? 'Sedang memindai…' : 'Siap'}</small>
      </div>
    </div>
    <button class="pill" onclick={onscan} disabled={scanning || health.scanning}>
      <Icon name="refresh" size={18} class={scanning || health.scanning ? 'spin' : ''} />
      <span>Pindai ulang</span>
    </button>
  </section>
  </div>

  <div class="col">
    <SharesList />
    <About />
  </div>
  </div>
</div>

<style>
  .settings {
    max-width: 560px;
    padding: 16px 16px 48px;
    display: flex;
    flex-direction: column;
    gap: 28px;
  }
  .cols, .col { display: flex; flex-direction: column; gap: 28px; }
  /* Layar lebar: dua kolom (koneksi/tampilan/pustaka | tautan/tentang) agar tidak satu kolom panjang di kiri. */
  @media (min-width: 1200px) {
    .settings { max-width: 1040px; padding: 24px 8px 56px; }
    .cols { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 56px; align-items: start; }
  }
  h1 { font-size: 22px; font-weight: 500; letter-spacing: -0.3px; color: var(--fg); }
  h2 { font-size: 13px; font-weight: 600; color: var(--mut); text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 12px; }
  section { display: flex; flex-direction: column; align-items: flex-start; gap: 12px; }
  section > h2 { align-self: stretch; margin-bottom: 0; }
  label { display: flex; flex-direction: column; gap: 6px; width: 100%; font-size: 13px; color: var(--fg-variant); }
  input {
    height: 42px;
    padding: 0 12px;
    border-radius: 10px;
    border: 1px solid var(--line);
    background: var(--bg);
    color: var(--fg);
    font: inherit;
    font-size: 14px;
  }
  input:focus { outline: 2px solid var(--acc); border-color: transparent; }
  small { color: var(--mut); font-size: 12px; }
  .msg { margin: 0; font-size: 13px; }
  .err { color: #d93025; }
  .ok { color: #188038; }
  .primary {
    height: 38px;
    padding: 0 20px;
    border-radius: 19px;
    background: var(--acc-bg);
    color: var(--acc-fg);
    font-weight: 500;
    font-size: 14px;
  }
  .primary:disabled { opacity: 0.5; }


  .stat { display: flex; align-items: center; gap: 12px; }
  .ico { width: 40px; height: 40px; border-radius: 12px; display: grid; place-items: center; background: var(--surface-variant); color: var(--mut); }
  .v { font-size: 15px; color: var(--fg); }
  .pill {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 38px;
    padding: 0 18px;
    border-radius: 19px;
    background: var(--acc-bg);
    color: var(--acc-fg);
    font-weight: 500;
    font-size: 14px;
  }
  .pill:disabled { opacity: 0.6; }
</style>
