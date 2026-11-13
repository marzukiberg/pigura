<script>
  import { onMount } from 'svelte';
  import Timeline from './lib/features/timeline/Timeline.svelte';
  import Explore from './lib/features/explore/Explore.svelte';
  import Settings from './lib/features/settings/Settings.svelte';
  import Duplicates from './lib/features/duplicates/Duplicates.svelte';
  import Login from './lib/features/auth/Login.svelte';
  import SharedView from './lib/features/share/SharedView.svelte';
  import SetupForm from './lib/features/shell/SetupForm.svelte';
  import AppBar from './lib/features/shell/AppBar.svelte';
  import NavBar from './lib/features/shell/NavBar.svelte';
  import Toast from './lib/ui/Toast.svelte';
  import { favoritesFetchUrl } from './lib/core/favorites.svelte.js';
  import { auth, checkAuth, installAuthHook } from './lib/core/auth.svelte.js';
  import { route, navigate, link } from './lib/core/router.svelte.js';
  import { titleState, buildTitle } from './lib/core/title.svelte.js';
  import { health, scanUi, checkHealth, triggerScan } from './lib/core/health.svelte.js';
  import { themeState, initTheme, setTheme } from './lib/core/theme.svelte.js';
  import { setup, initConnection, loadServerConfig, saveSettings } from './lib/core/connection.svelte.js';

  // Tampilan aktif ditentukan oleh URL (lihat lib/core/router.svelte.js).
  let r = $derived(route.current);
  let tab = $derived(r.view);
  let routePhotoId = $derived(r.photoId);
  let searchQuery = $state('');

  // Judul tab mengikuti halaman aktif (dan album/foto yang sedang dibuka).
  $effect(() => {
    document.title =
      (setup.configured && !setup.needsLibrary) || tab === 'share'
        ? buildTitle(tab, { photo: titleState.photo, album: titleState.album })
        : 'Pengaturan awal · pigura';
  });

  // Lightbox: /photos/:id di tab Foto, ?photo=ID di tab lain. Pindah antar foto
  // memakai replace agar tombol Kembali langsung menutup lightbox.
  function setPhotoRoute(id) {
    const base = tab === 'photos' ? '/' : route.path;
    const to = id ? (tab === 'photos' ? `/photos/${id}` : `${base}?photo=${id}`) : base;
    navigate(to, { replace: !!id && !!routePhotoId });
  }

  // Setelah login (token aktif) ambil folder library dari server.
  $effect(() => {
    if (setup.configured && auth.checked && auth.ok && !setup.scanBase) loadServerConfig();
  });

  onMount(() => {
    initTheme();
    installAuthHook();
    initConnection()
      .then(async () => {
        if (!setup.configured) return;
        await checkAuth();
        if (!auth.required || auth.ok) await loadServerConfig();
        checkHealth();
      });
    const interval = setInterval(() => setup.configured && checkHealth(), 5000);
    return () => clearInterval(interval);
  });
</script>

{#if r.view === 'share'}
  <!-- Tautan berbagi publik: berdiri sendiri, tanpa pengaturan awal/login pemilik -->
  <SharedView shareKey={r.shareKey} />
{:else if !setup.ready}
  <main class="boot" aria-busy="true"></main>
{:else if setup.publicOnly}
  <main class="boot public-only">
    <p>Halaman ini hanya untuk membuka tautan berbagi.</p>
  </main>
{:else if !setup.configured}
  <SetupForm />
{:else if !auth.checked}
  <main class="boot" aria-busy="true"></main>
{:else if auth.required && !auth.ok}
  <Login />
{:else if setup.needsLibrary}
  <SetupForm />
{:else}
  <div class="shell">
    <AppBar {tab} bind:searchQuery />
    <NavBar {tab} />

    <main class="content">
      {#if tab === 'photos'}
        <Timeline {searchQuery} photoId={routePhotoId} onphotochange={setPhotoRoute} />
      {:else if tab === 'explore'}
        <Explore bind:searchQuery photoId={routePhotoId} onphotochange={setPhotoRoute} />
      {:else if tab === 'albums'}
        {#await import('./lib/features/albums/Albums.svelte') then { default: Albums }}
          <Albums albumId={r.albumId} />
        {:catch}
          <p class="view-error">Album gagal dimuat. Coba buka lagi.</p>
        {/await}
      {:else if tab === 'favorites'}
        <Timeline
          fetchUrl={favoritesFetchUrl}
          filterFavorites={true}
          photoId={routePhotoId}
          onphotochange={setPhotoRoute}
          empty="Belum ada foto favorit. Ketuk bintang pada pratinjau foto untuk menjadikannya favorit."
        />
      {:else if tab === 'duplicates'}
        <Duplicates />
      {:else if tab === 'settings'}
        <Settings
          {health}
          theme={themeState.mode}
          scanning={scanUi.busy}
          saving={setup.saving}
          error={setup.error}
          saved={setup.saved}
          onscan={triggerScan}
          onsave={saveSettings}
          ontheme={setTheme}
        />
      {:else}
        <div class="not-found">
          <h2>Halaman tidak ditemukan</h2>
          <a href="/" onclick={link}>Kembali ke Foto</a>
        </div>
      {/if}
    </main>
  </div>
  <Toast />
{/if}

<style>
  .shell {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    background: var(--bg);
  }

  /* Main Content Layout */
  .content {
    flex: 1;
    padding: 0 4px calc(72px + env(safe-area-inset-bottom));
  }

  .boot { min-height: 100dvh; background: var(--bg); }
  .public-only { display: grid; place-items: center; padding: 24px; text-align: center; color: var(--muted, #888); }

  .not-found { padding: 64px 16px; text-align: center; color: var(--mut); }

  .not-found h2 { color: var(--fg); margin-bottom: 8px; }

  .not-found a { color: var(--acc); }

  .view-error { padding: 32px 16px; color: var(--mut); }

  @media (min-width: 900px) {
    .shell {
        display: grid;
        grid-template-columns: 240px 1fr;
        grid-template-rows: 64px 1fr;
      }
    .content {
        padding: 12px 24px 32px 16px;
        overflow-y: auto;
      }
  }

</style>
