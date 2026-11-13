<script>
  import Icon from '../../ui/Icon.svelte';
  import { link } from '../../core/router.svelte.js';
  import { health } from '../../core/health.svelte.js';

  let { tab } = $props();

  const TABS = [
    { id: 'photos', label: 'Foto', icon: 'photo', href: '/' },
    { id: 'explore', label: 'Jelajah', icon: 'explore', href: '/explore' },
    { id: 'albums', label: 'Album', icon: 'album', href: '/albums' },
    { id: 'favorites', label: 'Favorit', icon: 'star', href: '/favorites' },
    { id: 'duplicates', label: 'Duplikat', icon: 'copy', href: '/duplicates' },
    { id: 'settings', label: 'Pengaturan', icon: 'settings', href: '/settings' },
  ];
</script>

<!-- Navigasi: sidebar di desktop, bar bawah di ponsel -->
<nav class="navigation">
  <div class="nav-list">
    {#each TABS as t (t.id)}
      <a class="nav-item" class:active={tab === t.id} href={t.href} aria-current={tab === t.id ? 'page' : undefined} onclick={link}>
        <span class="nav-indicator">
          <Icon name={t.icon} size={22} strokeWidth={tab === t.id ? 2.5 : 2} />
        </span>
        <span class="nav-label">{t.label}</span>
      </a>
    {/each}
  </div>

  <!-- Widget penyimpanan (hanya sidebar desktop) -->
  <div class="sidebar-footer">
    <div class="storage-card">
      <div class="storage-icon"><Icon name="storage" size={18} /></div>
      <div class="storage-text">
        <div class="storage-title">Penyimpanan</div>
        <div class="storage-sub">
          {#if health.assets > 0}{health.assets} aset terindeks{:else}Memuat database…{/if}
        </div>
      </div>
    </div>
  </div>
</nav>

<style>
  .navigation {
    position: fixed;
    z-index: 25;
    inset: auto 0 0 0;
    background: var(--bar);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border-top: 1px solid var(--line);
    padding-bottom: env(safe-area-inset-bottom);
  }

  .nav-list {
    display: flex;
    justify-content: space-around;
    align-items: center;
    height: 64px;
  }

  .nav-item {
    text-decoration: none;
    color: inherit;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    color: var(--mut);
    font-size: 12px;
    font-weight: 500;
    transition: color 0.15s;
  }

  .nav-indicator {
    height: 32px;
    padding: 0 18px;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.2s, color 0.2s;
  }

  .nav-item.active {
    color: var(--fg);
  }

  .nav-item.active .nav-indicator {
    background: var(--acc-bg);
    color: var(--acc-fg);
  }

  .sidebar-footer {
    display: none;
  }

  @media (min-width: 900px) {
    .navigation {
        position: sticky;
        top: 64px;
        height: calc(100dvh - 64px);
        background: var(--bg);
        border-top: 0;
        border-right: 1px solid var(--line-subtle);
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        padding: 12px 12px 16px;
      }
    .nav-list {
        display: flex;
        flex-direction: column;
        gap: 4px;
        height: auto;
        width: 100%;
      }
    .nav-item {
        width: 100%;
        display: flex;
        flex: none;
        flex-direction: row;
        justify-content: flex-start;
        align-items: center;
        gap: 16px;
        height: 48px;
        padding: 0 16px;
        border-radius: 24px;
        font-size: 14px;
        font-weight: 500;
        color: var(--fg);
        box-sizing: border-box;
        transition: background 0.15s, color 0.15s;
      }
    .nav-indicator {
        padding: 0;
        width: 24px;
        height: 24px;
        background: none !important;
        color: inherit;
      }
    .nav-item:hover {
        background: var(--hover);
      }
    .nav-item.active {
        background: var(--acc-bg);
        color: var(--acc-fg);
        font-weight: 600;
      }
    .sidebar-footer {
        display: block;
        padding-top: 16px;
        border-top: 1px solid var(--line-subtle);
      }
    .storage-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        border-radius: 12px;
        background: var(--surface-variant);
      }
    .storage-icon {
        color: var(--mut);
        display: grid;
        place-items: center;
      }
    .storage-title {
        font-size: 12px;
        font-weight: 500;
        color: var(--fg);
      }
    .storage-sub {
        font-size: 11px;
        color: var(--mut);
      }
  }

  @media (max-width: 599px) {
    .nav-item[href='/settings'] { display: none; }
    .nav-indicator { padding: 0 14px; }
    .nav-label { font-size: 11px; }
  }
</style>
