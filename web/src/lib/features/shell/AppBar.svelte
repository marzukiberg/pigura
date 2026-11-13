<script>
  import Icon from '../../ui/Icon.svelte';
  import { link } from '../../core/router.svelte.js';
  import { health, scanUi, triggerScan } from '../../core/health.svelte.js';
  import { themeState, toggleTheme } from '../../core/theme.svelte.js';

  // tab: halaman aktif; searchQuery: teks pencarian (dibagi dengan daftar foto)
  let { tab, searchQuery = $bindable('') } = $props();
</script>

<!-- Bar atas: logo, pencarian, aksi cepat -->
<header class="app-bar">
  <div class="brand">
    <a href="/" class="brand-link" onclick={link}>
      <svg class="logo" viewBox="0 0 24 24" width="36" height="36" aria-hidden="true">
        <path fill="var(--google-red, #ea4335)" d="M12 2a5 5 0 0 1 5 5v5h-5a5 5 0 0 1-5-5 5 5 0 0 1 5-5z"/>
        <path fill="var(--google-yellow, #fbbc04)" d="M22 12a5 5 0 0 1-5 5h-5v-5a5 5 0 0 1 5-5 5 5 0 0 1 5 5z"/>
        <path fill="var(--google-green, #34a853)" d="M12 22a5 5 0 0 1-5-5v-5h5a5 5 0 0 1 5 5 5 5 0 0 1-5 5z"/>
        <path fill="var(--google-blue, #4285f4)" d="M2 12a5 5 0 0 1 5-5h5v5a5 5 0 0 1-5 5 5 5 0 0 1-5-5z"/>
      </svg>
      <span class="brand-name">Pigura</span>
    </a>
  </div>

  <div class="search-bar">
    <span class="search-icon"><Icon name="search" size={20} /></span>
    <input type="search" placeholder="Cari foto, tanggal, video" bind:value={searchQuery} aria-label="Cari foto" />
    {#if searchQuery}
      <button class="clear-btn" onclick={() => (searchQuery = '')} aria-label="Hapus pencarian">
        <Icon name="close" size={18} />
      </button>
    {/if}
  </div>

  <div class="actions">
    <button
      class="icon-btn"
      class:spin={scanUi.busy || health.scanning}
      onclick={triggerScan}
      title="Pindai ulang media library"
      aria-label="Pindai media"
    >
      <Icon name="refresh" size={20} />
    </button>

    <button class="icon-btn" onclick={toggleTheme} title="Ganti mode gelap/terang" aria-label="Mode tampilan">
      <Icon name={themeState.mode === 'dark' ? 'sun' : 'moon'} size={20} />
    </button>

    <a
      class="settings-btn"
      class:active={tab === 'settings'}
      href="/settings"
      onclick={link}
      title="Pengaturan"
      aria-label="Pengaturan"
      aria-current={tab === 'settings' ? 'page' : undefined}
    >
      <Icon name="settings" size={20} />
    </a>
  </div>
</header>

<style>
  .app-bar {
    position: sticky;
    top: 0;
    z-index: 30;
    height: calc(64px + env(safe-area-inset-top));
    padding: env(safe-area-inset-top) 16px 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    background: var(--surface);
    border-bottom: 1px solid var(--line-subtle);
  }

  .brand {
    display: flex;
    align-items: center;
    flex: none;
  }

  .brand-link {
    display: flex;
    align-items: center;
    gap: 8px;
    text-decoration: none;
    color: var(--fg);
  }

  .logo {
    display: block;
    width: 36px;
    height: 36px;
  }

  .brand-name {
    font-size: 20px;
    font-weight: 500;
    letter-spacing: -0.3px;
    color: var(--fg);
  }

  .search-bar {
    flex: 1;
    max-width: 720px;
    height: 48px;
    background: var(--search-bg);
    border-radius: 24px;
    display: flex;
    align-items: center;
    padding: 0 16px;
    gap: 12px;
    transition: background 0.2s, box-shadow 0.2s;
  }

  .search-bar:focus-within {
    background: var(--search-focus-bg);
    box-shadow: var(--card-shadow);
  }

  .search-icon {
    display: flex;
    align-items: center;
    color: var(--mut);
    flex: none;
  }

  .search-bar input {
    flex: 1;
    height: 100%;
    background: transparent;
    border: 0;
    outline: none;
    color: var(--fg);
    font-size: 15px;
  }

  .search-bar input::placeholder {
    color: var(--mut);
  }

  .clear-btn {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    color: var(--mut);
    display: grid;
    place-items: center;
  }

  .clear-btn:hover {
    background: var(--hover);
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: none;
  }

  .icon-btn {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: var(--mut);
    transition: background 0.15s, color 0.15s;
  }

  .icon-btn:hover {
    background: var(--hover);
    color: var(--fg);
  }

  .icon-btn.spin :global(svg) {
    animation: rotate 1s linear infinite;
  }

  @keyframes rotate {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .settings-btn {
    width: 38px;
    height: 38px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    background: var(--acc-bg);
    color: var(--acc-fg);
    text-decoration: none;
    user-select: none;
    transition: filter 0.15s, box-shadow 0.15s;
  }

  .settings-btn:hover { filter: brightness(1.1); }

  .settings-btn.active { box-shadow: inset 0 0 0 2px var(--acc); }

  @media (min-width: 900px) {
    .app-bar {
        grid-column: 1 / 3;
        padding: env(safe-area-inset-top) 24px 0;
      }
  }

  @media (max-width: 599px) {
    .app-bar {
        padding-left: 12px;
        padding-right: 12px;
        gap: 10px;
      }
    .brand-name,
    .actions .icon-btn { display: none; }
    .search-bar { height: 44px; padding: 0 14px; gap: 10px; }
  }
</style>
