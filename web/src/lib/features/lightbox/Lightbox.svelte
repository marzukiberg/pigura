<script>
  import { titleState } from '../../core/title.svelte.js';
  import Icon from '../../ui/Icon.svelte';
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte';
  import ShareDialog from '../share/ShareDialog.svelte';
  import InfoPanel from './InfoPanel.svelte';
  import { api } from '../../core/api.svelte.js';
  import { dateFmt, baseName } from '../../core/format.js';

  let { items, index = $bindable(0), onclose, onchange, ontogglefav, ondelete, favorites } = $props();

  let a = $derived(items[index]);

  // Judul tab = nama file foto yang sedang dibuka
  $effect(() => {
    titleState.photo = a ? a.path.split('/').pop() : '';
    return () => (titleState.photo = '');
  });
  let startX = 0;
  let showInfo = $state(false);
  let zoomed = $state(false);
  let showDeleteConfirm = $state(false);
  let showShare = $state(false);
  let deleting = $state(false);

  let fullLoaded = $state(false);
  let fullError = $state(false);
  let thumbError = $state(false);
  let usePreviewFallback = $state(false);

  // Reset image loading states when current item changes
  $effect(() => {
    if (a) {
      fullLoaded = false;
      fullError = false;
      thumbError = false;
      usePreviewFallback = false;
      zoomed = false;
    }
  });

  const fmt = dateFmt.full;
  let when = $derived(a ? fmt.format(new Date(a.taken_at * 1000)) : '');

  const go = (d) => {
    const n = index + d;
    if (n >= 0 && n < items.length) {
      index = n;
      onchange?.(items[n].id);
    }
  };

  function key(e) {
    if (showShare) return; // dialog berbagi menangani tombolnya sendiri
    if (e.key === 'Escape') {
      if (showInfo) showInfo = false;
      else onclose();
    } else if (e.key === 'ArrowRight') {
      go(1);
    } else if (e.key === 'ArrowLeft') {
      go(-1);
    } else if (e.key.toLowerCase() === 'i') {
      showInfo = !showInfo;
    }
  }

  function down(e) {
    if (e.target.closest('.drawer') || e.target.closest('.top') || e.target.closest('.nav')) return;
    startX = e.clientX;
  }
  function up(e) {
    if (e.target.closest('.drawer') || e.target.closest('.top') || e.target.closest('.nav')) return;
    const dx = e.clientX - startX;
    if (Math.abs(dx) > 60) go(dx < 0 ? 1 : -1);
  }


  const getFilename = (path) => baseName(path) || 'Foto';



  // Browser tidak mendukung HEIC/HEIF secara luas. Foto yang browser dukung
  // selalu memakai endpoint original; preview hanya fallback untuk format itu.
  function needsPreview(path) {
    return !/\.(avif|gif|jpe?g|png|webp)$/i.test(path || '');
  }

  let fullSrc = $derived(
    a &&
      api(
        `/api/assets/${a.id}/${usePreviewFallback || needsPreview(a.path) ? 'preview' : 'original'}${a.taken_at ? `?t=${a.taken_at}` : ''}`
      )
  );

  function onFullError() {
    if (!usePreviewFallback && !needsPreview(a?.path)) {
      usePreviewFallback = true;
      return;
    }
    fullError = true;
  }

  function checkComplete(node) {
    if (node.complete && node.naturalWidth > 0) {
      fullLoaded = true;
    }
  }

  async function handleDeletePhoto() {
    if (!a || deleting) return;
    deleting = true;
    const targetId = a.id;
    try {
      const res = await fetch(api(`/api/assets/${targetId}`), { method: 'DELETE' });
      if (!res.ok) throw new Error('Gagal menghapus foto.');
      showDeleteConfirm = false;
      ondelete?.(targetId);
    } catch (err) {
      alert(err.message || 'Gagal menghapus foto.');
    } finally {
      deleting = false;
    }
  }
</script>

<svelte:window onkeydown={key} />

<div class="lb" role="dialog" aria-modal="true" tabindex="-1" onpointerdown={down} onpointerup={up}>
  <!-- Top App Bar Google Photos Lightbox -->
  <div class="top">
    <div class="top-left">
      <button class="ib" onclick={onclose} aria-label="Kembali" title="Kembali (Esc)">
        <Icon name="back" size={22} />
      </button>
      <span class="when">{when}</span>
    </div>

    <div class="top-right">
      {#if a}
        <button
          class="ib"
          class:active={zoomed}
          onclick={() => (zoomed = !zoomed)}
          aria-label="Perbesar"
          title="Zoom"
        >
          <Icon name={zoomed ? 'zoomOut' : 'zoomIn'} size={22} />
        </button>

        {#if ontogglefav && favorites}
          <button
            class="ib"
            class:starred={favorites.has(a.id)}
            onclick={() => ontogglefav(a.id)}
            aria-label="Favorit"
            title="Favorit"
          >
            <Icon name={favorites.has(a.id) ? 'starFilled' : 'star'} size={22} />
          </button>
        {/if}

        <a
          class="ib"
            href={api(`/api/assets/${a.id}/original`)}
          download={getFilename(a.path)}
          aria-label="Unduh file asli"
          title="Unduh asli"
        >
          <Icon name="download" size={22} />
        </a>

        <button class="ib" onclick={() => (showShare = true)} aria-label="Bagikan" title="Bagikan">
          <Icon name="share" size={22} />
        </button>

        <button
          class="ib btn-trash"
          onclick={() => (showDeleteConfirm = true)}
          aria-label="Hapus foto"
          title="Hapus foto"
        >
          <Icon name="trash" size={22} />
        </button>

        <button
          class="ib"
          class:active={showInfo}
          onclick={() => (showInfo = !showInfo)}
          aria-label="Informasi file"
          title="Detail info (i)"
        >
          <Icon name="info" size={22} />
        </button>
      {/if}
    </div>
  </div>

  <!-- Stage Area Media -->
  {#if a}
    {#key a.id}
      <div class="stage" class:zoomed class:with-drawer={showInfo}>
        {#if a.type === 'video'}
          <!-- svelte-ignore a11y_media_has_caption -->
          <video src={api(`/api/assets/${a.id}/original`)} controls autoplay playsinline></video>
        {:else}
          <!-- Layer 1: Low-res thumbnail ditampilkan cepat -->
          {#if !thumbError}
            <img
              class="ph"
              class:hidden={fullLoaded}
                src={api(`/api/assets/${a.id}/thumb${a.taken_at ? `?t=${a.taken_at}` : ''}`)}
              alt=""
              onerror={() => (thumbError = true)}
            />
          {/if}

          <!-- Layer 2: file asli untuk format yang didukung browser. -->
          {#if !fullError}
            <img
              class="full"
              class:visible={fullLoaded}
                src={fullSrc}
                alt=""
                decoding="async"
                fetchpriority="high"
                onload={() => (fullLoaded = true)}
                onerror={onFullError}
                use:checkComplete
            />
          {/if}

          <!-- Fallback jika gagal muat file sama sekali (bebas ikon broken bawaan) -->
          {#if fullError && thumbError}
            <div class="error-placeholder">
              <Icon name="brokenImage" size={48} />
              <p>Tidak dapat memuat pratinjau gambar</p>
              <a href={api(`/api/assets/${a.id}/original`)} download class="retry-link">Unduh file asli</a>
            </div>
          {/if}

          <!-- Loading subtle spinner saat preview sedang dimuat -->
          {#if !fullLoaded && !fullError && a.type !== 'video'}
            <div class="loading-ring">
              <div class="spinner"></div>
            </div>
          {/if}
        {/if}
      </div>
    {/key}
  {/if}

  <!-- Navigasi Kiri & Kanan -->
  {#if index > 0}
    <button class="nav l" onclick={() => go(-1)} aria-label="Sebelumnya" title="Sebelumnya (←)">
      <Icon name="left" size={28} />
    </button>
  {/if}
  {#if index < items.length - 1}
    <button class="nav r" onclick={() => go(1)} aria-label="Berikutnya" title="Berikutnya (→)">
      <Icon name="right" size={28} />
    </button>
  {/if}

  {#if showInfo && a}
    <InfoPanel {a} {when} onclose={() => (showInfo = false)} />
  {/if}

  {#if showShare && a}
    <ShareDialog kind="asset" targetId={a.id} onclose={() => (showShare = false)} />
  {/if}

  <!-- Modal Konfirmasi Hapus Cantik -->
  <ConfirmDialog
    open={showDeleteConfirm}
    title="Hapus foto?"
    message="Foto ini akan disembunyikan dari Pigura. File asli di disk tidak dihapus."
    confirmLabel="Hapus"
    cancelLabel="Batal"
    loading={deleting}
    onconfirm={handleDeletePhoto}
    oncancel={() => (showDeleteConfirm = false)}
  />
</div>

<style>
  .lb {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: #0b0b0c;
    color: #ffffff;
    touch-action: pan-y;
    overflow: hidden;
  }

  .stage {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    transition: right 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .stage.with-drawer {
    right: 360px;
  }

  @media (max-width: 768px) {
    .stage.with-drawer {
      right: 0;
    }
  }

  .stage img,
  .stage video {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    max-width: 100%;
    max-height: 100%;
    min-width: 0;
    min-height: 0;
    object-fit: contain;
    object-position: center;
    image-orientation: from-image;
    user-select: none;
    -webkit-user-select: none;
    transition: transform 0.2s ease-out;
  }

  .stage.zoomed img,
  .stage.zoomed video {
    transform: scale(2);
    cursor: zoom-out;
  }

  .ph {
    filter: none;
    opacity: 1;
    transition: opacity 0.2s ease-out;
  }

  .ph.hidden {
    opacity: 0;
    pointer-events: none;
  }

  .full {
    opacity: 0;
    transition: opacity 0.2s ease-in-out;
  }

  .full.visible {
    opacity: 1;
  }

  .error-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: #9aa0a6;
    padding: 24px;
    text-align: center;
  }

  .retry-link {
    color: var(--acc, #8ab4f8);
    font-size: 14px;
    text-decoration: underline;
  }

  /* Loading spinner halus di pojok bawah */
  .loading-ring {
    position: absolute;
    bottom: 24px;
    left: 24px;
    z-index: 10;
    width: 28px;
    height: 28px;
    pointer-events: none;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 3px solid rgba(255, 255, 255, 0.2);
    border-top-color: #ffffff;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Top Bar Google Photos */
  .top {
    position: absolute;
    z-index: 15;
    inset: 0 0 auto 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: calc(12px + env(safe-area-inset-top)) 16px 28px;
    background: linear-gradient(rgba(0, 0, 0, 0.75), transparent);
  }

  .top-left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .when {
    font-size: 15px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .top-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .ib {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: #ffffff;
    text-decoration: none;
    transition: background 0.15s, color 0.15s;
  }

  .ib:hover {
    background: rgba(255, 255, 255, 0.15);
  }

  .ib.active {
    background: rgba(255, 255, 255, 0.25);
    color: var(--acc, #8ab4f8);
  }

  .ib.starred {
    color: #fbbc04;
  }


  /* Floating Left/Right Nav Buttons */
  .nav {
    position: absolute;
    z-index: 10;
    top: 50%;
    transform: translateY(-50%);
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: #ffffff;
    background: rgba(255, 255, 255, 0.12);
    backdrop-filter: blur(8px);
    opacity: 0.8;
    transition: opacity 0.15s, background 0.15s, transform 0.15s;
  }

  .nav:hover {
    opacity: 1;
    background: rgba(255, 255, 255, 0.25);
    transform: translateY(-50%) scale(1.05);
  }

  .l { left: 16px; }
  .r { right: 16px; }

  @media (pointer: coarse) {
    .nav { display: none; }
  }

  @media (max-width: 480px) {
    .top { padding-left: 6px; padding-right: 6px; gap: 4px; }
    .top-left { gap: 4px; }
    .top-right { gap: 0; }
    .ib { width: 40px; height: 40px; }
    .when { display: none; } /* tanggal ada di panel info */
  }
</style>
