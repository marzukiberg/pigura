<script>
  import { onMount } from 'svelte';
  import Icon from '../../ui/Icon.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { createThumbState } from '../../ui/thumb.svelte.js';
  import { dateFmt } from '../../core/format.js';
  import { api } from '../../core/api.svelte.js';
  import { titleState } from '../../core/title.svelte.js';

  // Halaman pengunjung publik (/s/<kunci>): tanpa shell aplikasi, tanpa login pemilik.
  let { shareKey } = $props();

  let state = $state('loading'); // loading | missing | password | ready | error
  let meta = $state(null);
  let items = $state([]);
  let allowDownload = $state(false);
  let pw = $state('');
  let pwError = $state('');
  let busy = $state(false);
  let open = $state(-1);
  const thumbs = createThumbState();

  const base = $derived(`/api/s/${shareKey}`);
  const media = (id, what) => api(`${base}/a/${id}/${what}`);
  const fmt = dateFmt.long;

  async function loadItems() {
    const res = await fetch(api(`${base}/items`));
    if (res.status === 401) return (state = 'password');
    if (!res.ok) return (state = res.status === 404 ? 'missing' : 'error');
    const d = await res.json();
    items = d.items;
    allowDownload = d.allow_download;
    state = 'ready';
    titleState.album = meta?.title || '';
    if (meta?.kind === 'asset' && items.length === 1) open = 0;
  }

  async function init() {
    try {
      const res = await fetch(api(base));
      if (res.status === 404) return (state = 'missing');
      if (!res.ok) return (state = 'error');
      meta = await res.json();
      if (meta.requires_password && !meta.unlocked) return (state = 'password');
      await loadItems();
    } catch {
      state = 'error';
    }
  }

  async function unlock(e) {
    e.preventDefault();
    if (!pw || busy) return;
    busy = true;
    pwError = '';
    try {
      const res = await fetch(api(`${base}/unlock`), { method: 'POST', body: JSON.stringify({ password: pw }) });
      if (res.ok) {
        const m = await fetch(api(base));
        if (m.ok) meta = await m.json();
        await loadItems();
      } else {
        pwError = res.status === 429 ? 'Terlalu banyak percobaan. Coba lagi beberapa menit lagi.' : 'Password salah.';
      }
    } catch {
      pwError = 'Tidak bisa menghubungi server.';
    } finally {
      busy = false;
    }
  }

  function key(e) {
    if (open < 0) return;
    if (e.key === 'Escape') close();
    else if (e.key === 'ArrowRight') step(1);
    else if (e.key === 'ArrowLeft') step(-1);
  }
  const step = (d) => (open = Math.min(items.length - 1, Math.max(0, open + d)));
  const close = () => (open = -1);
  let sx = 0;
  const down = (e) => (sx = e.clientX);
  const up = (e) => {
    const dx = e.clientX - sx;
    if (Math.abs(dx) > 60) step(dx < 0 ? 1 : -1);
  };

  onMount(() => {
    const m = document.createElement('meta');
    m.name = 'robots';
    m.content = 'noindex, nofollow';
    document.head.appendChild(m);
    init();
    return () => {
      m.remove();
      titleState.album = '';
    };
  });
</script>

<svelte:window onkeydown={key} />

<main class="page">
  {#if state === 'loading'}
    <p class="msg" aria-busy="true">Memuat…</p>
  {:else if state === 'missing'}
    <div class="msg center">
      <Icon name="brokenImage" size={48} />
      <h1>Tautan tidak ditemukan</h1>
      <p>Tautan ini tidak ada, sudah kedaluwarsa, atau sudah dicabut oleh pemiliknya.</p>
    </div>
  {:else if state === 'error'}
    <div class="msg center">
      <Icon name="brokenImage" size={48} />
      <h1>Gagal memuat</h1>
      <button class="go" onclick={init}>Coba lagi</button>
    </div>
  {:else if state === 'password'}
    <form class="card" onsubmit={unlock}>
      <div class="mark"><Icon name="lock" size={26} /></div>
      <h1>Dilindungi password</h1>
      <p>Masukkan password untuk melihat foto yang dibagikan.</p>
      <input type="password" bind:value={pw} autocomplete="current-password" placeholder="Password" aria-label="Password" aria-invalid={!!pwError} />
      {#if pwError}<p class="err" role="alert">{pwError}</p>{/if}
      <button class="go" disabled={!pw || busy}>{busy ? 'Memeriksa…' : 'Buka'}</button>
    </form>
  {:else}
    <header>
      <h1>{meta?.title || 'Foto dibagikan'}</h1>
      <span class="sub">
        {items.length} item{meta?.expires_at ? ` · berlaku sampai ${fmt.format(new Date(meta.expires_at * 1000))}` : ''}
      </span>
    </header>

    {#if items.length === 0}
      <p class="msg">Tidak ada foto di sini.</p>
    {:else}
      <div class="grid">
        {#each items as a, i (a.id)}
          <button class="cell" onclick={() => (open = i)} aria-label="Buka">
            <Thumb src={media(a.id, 'thumb')} id={a.id} state={thumbs} />
            {#if a.type === 'video'}<span class="vid"><Icon name="play" size={12} /></span>{/if}
          </button>
        {/each}
      </div>
    {/if}
    <footer>Dibagikan lewat pigura</footer>
  {/if}
</main>

{#if open >= 0 && items[open]}
  {@const a = items[open]}
  <div class="lb" role="dialog" aria-modal="true" onpointerdown={down} onpointerup={up}>
    <div class="top">
      <button class="ib" onclick={close} aria-label="Tutup"><Icon name="close" size={22} /></button>
      <span class="count">{open + 1} / {items.length}</span>
      {#if allowDownload}
        <a class="ib" href={media(a.id, 'original') + '?dl=1'} aria-label="Unduh" download><Icon name="download" size={22} /></a>
      {:else}<span class="ib"></span>{/if}
    </div>
    {#key a.id}
      {#if a.type === 'video'}
        <!-- svelte-ignore a11y_media_has_caption -->
        <video src={media(a.id, 'original')} controls playsinline autoplay></video>
      {:else}
        <img class="full" src={media(a.id, 'preview')} alt="" draggable="false" />
      {/if}
    {/key}
    {#if open > 0}<button class="nav l" onclick={() => step(-1)} aria-label="Sebelumnya"><Icon name="left" size={24} /></button>{/if}
    {#if open < items.length - 1}<button class="nav r" onclick={() => step(1)} aria-label="Berikutnya"><Icon name="right" size={24} /></button>{/if}
  </div>
{/if}

<style>
  .page { min-height: 100dvh; background: var(--bg); color: var(--fg); padding: 16px 8px calc(24px + env(safe-area-inset-bottom)); }
  header { padding: 8px 8px 16px; display: flex; flex-direction: column; gap: 2px; }
  h1 { font-size: 22px; font-weight: 500; letter-spacing: -0.3px; }
  .sub { font-size: 13px; color: var(--mut); }
  .msg { color: var(--mut); text-align: center; padding: 48px 16px; margin: 0; }
  .center { display: flex; flex-direction: column; align-items: center; gap: 12px; padding-top: 18vh; }
  .center h1 { color: var(--fg); }
  .center p { margin: 0; max-width: 340px; font-size: 14px; line-height: 1.5; }
  .card { margin: 18vh auto 0; width: min(100%, 380px); padding: 28px; border-radius: 24px; background: var(--surface); border: 1px solid var(--line); box-shadow: var(--card-shadow); display: flex; flex-direction: column; gap: 14px; }
  .card p { margin: 0; font-size: 13px; color: var(--mut); line-height: 1.45; }
  .mark { width: 48px; height: 48px; border-radius: 14px; display: grid; place-items: center; background: var(--acc-bg); color: var(--acc-fg); }
  input { height: 46px; padding: 0 14px; border-radius: 12px; border: 1px solid var(--line); background: var(--bg); color: var(--fg); font: inherit; }
  input:focus { outline: 2px solid var(--acc); border-color: transparent; }
  input[aria-invalid='true'] { border-color: #d93025; }
  .err { color: #d93025 !important; }
  .go { height: 44px; padding: 0 22px; border-radius: 22px; background: var(--acc-bg); color: var(--acc-fg); font-weight: 500; font-size: 15px; }
  .go:disabled { opacity: 0.5; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(112px, 1fr)); gap: 4px; }
  @media (min-width: 700px) { .grid { grid-template-columns: repeat(auto-fill, minmax(168px, 1fr)); } }
  .cell { position: relative; aspect-ratio: 1; overflow: hidden; background: var(--tile); padding: 0; border-radius: 4px; }
  .vid { position: absolute; right: 6px; bottom: 6px; width: 22px; height: 22px; border-radius: 50%; display: grid; place-items: center; background: rgba(0, 0, 0, 0.6); color: #fff; }
  footer { text-align: center; font-size: 12px; color: var(--mut); padding: 28px 0 8px; }

  .lb { position: fixed; inset: 0; z-index: 100; background: #0b0b0c; color: #fff; overflow: hidden; touch-action: pan-y; }
  /* Penuhi viewer secara eksplisit (bukan height:100% pada baris grid auto, yang jatuh ke ukuran alami
     gambar dan membuat foto potret meluap di layar lebar). contain = seluruh foto terlihat. */
  .lb .full, .lb video { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; object-position: center; }
  .top { position: absolute; z-index: 5; inset: 0 0 auto 0; display: flex; align-items: center; justify-content: space-between; padding: calc(10px + env(safe-area-inset-top)) 10px 24px; background: linear-gradient(rgba(0, 0, 0, 0.7), transparent); }
  .ib { width: 44px; height: 44px; border-radius: 50%; display: grid; place-items: center; color: #fff; text-decoration: none; }
  .ib:hover { background: rgba(255, 255, 255, 0.15); }
  .count { font-size: 14px; opacity: 0.85; }
  .nav { position: absolute; top: 50%; transform: translateY(-50%); width: 48px; height: 48px; border-radius: 50%; display: grid; place-items: center; color: #fff; background: rgba(255, 255, 255, 0.12); }
  .nav.l { left: 12px; } .nav.r { right: 12px; }
  @media (pointer: coarse) { .nav { display: none; } }
</style>
