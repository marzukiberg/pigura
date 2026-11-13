<script>
  import { onMount } from 'svelte';
  import { ContextMenu } from 'bits-ui';
  import Timeline from '../timeline/Timeline.svelte';
  import { api } from '../../core/api.svelte.js';
  import Icon from '../../ui/Icon.svelte';
  import PhotoPicker from './PhotoPicker.svelte';
  import Seg from '../../ui/Seg.svelte';
  import RenameDialog from '../../ui/RenameDialog.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { createThumbState } from '../../ui/thumb.svelte.js';
  import { showToast } from '../../core/toast.svelte.js';
  import ShareDialog from '../share/ShareDialog.svelte';
  import { navigate, link } from '../../core/router.svelte.js';
  import { titleState } from '../../core/title.svelte.js';

  // albumId dari route /albums/:id (null = daftar album)
  let { albumId = null } = $props();

  let albums = $state([]);
  let ready = $state(false);
  let name = $state('');
  let creating = $state(false);
  let current = $derived(albumId ? (albums.find((a) => a.id === albumId) ?? null) : null);
  $effect(() => {
    titleState.album = current?.name ?? '';
    return () => (titleState.album = '');
  });
  let adding = $state(false);
  let sharing = $state(false);
  let reloadKey = $state(0);
  let segment = $state('all');
  let renaming = $state(null); // album yang sedang diganti namanya

  const SEGMENTS = [
    { id: 'all', label: 'Semua' },
    { id: 'filled', label: 'Berisi' },
    { id: 'empty', label: 'Kosong' },
  ];
  let shown = $derived(
    albums.filter((a) => (segment === 'filled' ? a.count > 0 : segment === 'empty' ? a.count === 0 : true))
  );
  let segCount = $derived({
    all: albums.length,
    filled: albums.filter((a) => a.count > 0).length,
    empty: albums.filter((a) => a.count === 0).length,
  });

  function photosAdded(n) {
    adding = false;
    current.count += n;
    reloadKey++;
    showToast(n > 0 ? `${n} foto ditambahkan ke “${current.name}”` : 'Semua foto itu sudah ada di album', 2600);
  }
  const thumbs = createThumbState();

  async function load() {
    try {
      albums = await (await fetch(api('/api/albums'))).json();
    } catch {
      albums = [];
    } finally {
      ready = true;
    }
  }

  async function create(e) {
    e.preventDefault();
    const n = name.trim();
    if (!n) return;
    await fetch(api('/api/albums'), { method: 'POST', body: JSON.stringify({ name: n }) });
    name = '';
    creating = false;
    load();
  }

  async function doRename(newName) {
    const al = renaming;
    const res = await fetch(api(`/api/albums/${al.id}`), { method: 'PATCH', body: JSON.stringify({ name: newName }) });
    if (!res.ok) throw new Error(res.status === 404 ? 'Album tidak ditemukan.' : 'Gagal mengganti nama album.');
    const { name } = await res.json();
    const target = albums.find((a) => a.id === al.id);
    if (target) target.name = name;
    renaming = null;
    showToast(`Album diganti nama menjadi “${name}”`, 2400);
  }

  onMount(load);
  // muat ulang daftar (jumlah item) saat kembali dari detail album
  let prevAlbumId = albumId;
  $effect(() => {
    const id = albumId;
    if (prevAlbumId && !id) load();
    prevAlbumId = id;
  });
</script>

{#snippet albumMenu(al)}
  <ContextMenu.Portal>
    <ContextMenu.Content class="ctx-menu">
      <ContextMenu.Item class="ctx-item" onSelect={() => (renaming = al)}>
        <Icon name="edit" size={18} />
        <span>Ganti nama</span>
      </ContextMenu.Item>
    </ContextMenu.Content>
  </ContextMenu.Portal>
{/snippet}

<RenameDialog
  open={!!renaming}
  title="Ganti nama album"
  value={renaming?.name ?? ''}
  onsubmit={doRename}
  oncancel={() => (renaming = null)}
/>

{#if albumId && !current}
  <div class="albums-page">
    <button class="ib" onclick={() => navigate('/albums')} aria-label="Kembali ke album"><Icon name="back" size={22} /></button>
    {#if ready}
      <div class="empty-state"><Icon name="album" size={48} /><p>Album tidak ditemukan</p></div>
    {/if}
  </div>
{:else if current}
  <div class="album-view">
    <div class="album-bar">
      <button class="ib" onclick={() => navigate('/albums')} aria-label="Kembali ke album">
        <Icon name="back" size={22} />
      </button>
      <ContextMenu.Root>
        <ContextMenu.Trigger>
          {#snippet child({ props })}
            <div {...props} class="album-meta">
              <h1>{current.name}</h1>
              <span class="count-label">{current.count} item</span>
            </div>
          {/snippet}
        </ContextMenu.Trigger>
        {@render albumMenu(current)}
      </ContextMenu.Root>
      <button class="ib" onclick={() => (renaming = current)} aria-label="Ganti nama album" title="Ganti nama album">
        <Icon name="edit" size={20} />
      </button>
      <button class="ib" onclick={() => (sharing = true)} aria-label="Bagikan album" title="Bagikan album">
        <Icon name="share" size={20} />
      </button>
      <button class="create-pill add-photo" onclick={() => (adding = true)}>
        <Icon name="add" size={18} />
        <span>Tambah foto</span>
      </button>
    </div>
    {#key `${current.id}-${reloadKey}`}
      <Timeline serverOrder={false} fetchUrl={() => api(`/api/albums/${current.id}/assets`)} empty="Album kosong. Ketuk “Tambah foto” untuk mengisinya." />
    {/key}
  </div>
  {#if sharing}
    <ShareDialog kind="album" targetId={current.id} title={current.name} onclose={() => (sharing = false)} />
  {/if}
  {#if adding}
    <PhotoPicker albumId={current.id} albumName={current.name} onclose={() => (adding = false)} ondone={photosAdded} />
  {/if}
{:else}
  <div class="albums-page">
    <div class="albums-header">
      <h1>Album</h1>
      {#if !creating}
        <button class="create-pill" onclick={() => (creating = true)}>
          <Icon name="add" size={18} />
          <span>Buat album</span>
        </button>
      {/if}
    </div>

    {#if albums.length > 0}
      <div class="segwrap">
        <Seg
          role="tab"
          variant="inline"
          size="sm"
          label="Saring album"
          options={SEGMENTS.map((sg) => ({ ...sg, count: segCount[sg.id] }))}
          value={segment}
          onchange={(v) => (segment = v)}
        />
      </div>
    {/if}

    <div class="grid">
      {#if creating}
        <form class="card new-card" onsubmit={create}>
          <div class="new-cover">
            <Icon name="album" size={40} />
          </div>
          <input bind:value={name} placeholder="Judul album" />
          <div class="form-actions">
            <button type="button" class="btn-cancel" onclick={() => (creating = false)}>Batal</button>
            <button class="btn-submit" disabled={!name.trim()}>Simpan</button>
          </div>
        </form>
      {/if}

      {#each shown as al (al.id)}
        <ContextMenu.Root>
          <ContextMenu.Trigger>
            {#snippet child({ props })}
              <a {...props} class="card" href={`/albums/${al.id}`} onclick={link}>
                <span class="cover">
                  {#if al.cover_id}
                    <Thumb src={api(`/api/assets/${al.cover_id}/thumb`)} id={al.id} state={thumbs}>
                      {#snippet fallback()}<span class="cover-icon"><Icon name="album" size={40} /></span>{/snippet}
                    </Thumb>
                  {:else}
                    <span class="cover-icon"><Icon name="album" size={40} /></span>
                  {/if}
                </span>
                <span class="nm">{al.name}</span>
                <span class="sub">{al.count} item</span>
              </a>
            {/snippet}
          </ContextMenu.Trigger>
          {@render albumMenu(al)}
        </ContextMenu.Root>
      {/each}
    </div>

    {#if ready && albums.length > 0 && shown.length === 0 && !creating}
      <div class="empty-state">
        <Icon name="album" size={48} />
        <p>{segment === 'empty' ? 'Tidak ada album kosong' : 'Belum ada album berisi'}</p>
      </div>
    {/if}

    {#if ready && albums.length === 0 && !creating}
      <div class="empty-state">
        <Icon name="album" size={48} />
        <p>Belum ada album</p>
        <button class="create-pill" onclick={() => (creating = true)}>
          <Icon name="add" size={18} />
          <span>Buat album pertama</span>
        </button>
      </div>
    {/if}
  </div>
{/if}

<style>
  :global(.ctx-menu) {
    z-index: 250;
    min-width: 180px;
    padding: 6px;
    border-radius: 14px;
    background: var(--surface);
    color: var(--fg);
    border: 1px solid var(--line);
    box-shadow: 0 8px 28px rgba(0, 0, 0, 0.3);
    outline: none;
  }
  :global(.ctx-item) {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 38px;
    padding: 0 12px;
    border-radius: 9px;
    font-size: 14px;
    cursor: pointer;
    user-select: none;
    outline: none;
  }
  :global(.ctx-item[data-highlighted]) { background: var(--hover); }

  .albums-page {
    padding: 12px 12px 32px;
  }

  .albums-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
    padding: 0 4px;
  }

  h1 {
    font-size: 22px;
    font-weight: 500;
    color: var(--fg);
    letter-spacing: -0.3px;
  }

  .create-pill {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 40px;
    padding: 0 18px;
    border-radius: 20px;
    background: var(--acc-bg);
    color: var(--acc-fg);
    font-size: 14px;
    font-weight: 500;
    transition: background 0.15s, filter 0.15s;
  }

  .create-pill:hover {
    filter: brightness(0.96);
  }

  .segwrap { margin: 0 4px 20px; }

  .add-photo { margin-left: 0; }
  .album-bar .ib:first-of-type { margin-left: 0; }
  .album-meta + .ib { margin-left: auto; }


  .album-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 4px 16px;
  }

  .album-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    flex: 1;
  }

  .album-meta h1 {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .count-label {
    font-size: 13px;
    color: var(--mut);
  }

  .ib {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: var(--mut);
    transition: background 0.15s, color 0.15s;
  }

  .ib:hover {
    background: var(--hover);
    color: var(--fg);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 20px;
  }

  .card {
    color: inherit;
    text-decoration: none;
    display: flex;
    flex-direction: column;
    text-align: left;
    min-width: 0;
    background: transparent;
    transition: transform 0.15s ease-out;
  }

  .card:hover {
    transform: translateY(-2px);
  }

  .cover {
    aspect-ratio: 1;
    border-radius: 16px;
    overflow: hidden;
    background: var(--tile);
    display: grid;
    place-items: center;
    color: var(--mut);
    margin-bottom: 8px;
    box-shadow: var(--card-shadow);
    transition: filter 0.15s;
  }

  .cover-icon { display: grid; place-items: center; width: 100%; height: 100%; color: var(--mut); }

  .card:hover .cover {
    filter: brightness(0.95);
  }

  .nm {
    font-size: 15px;
    font-weight: 500;
    color: var(--fg);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sub {
    font-size: 13px;
    color: var(--mut);
  }

  .new-card {
    border: 2px dashed var(--line);
    border-radius: 16px;
    padding: 12px;
    background: var(--surface);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .new-cover {
    aspect-ratio: 1.6;
    border-radius: 12px;
    background: var(--surface-variant);
    display: grid;
    place-items: center;
    color: var(--acc);
  }

  input {
    height: 40px;
    padding: 0 12px;
    border-radius: 8px;
    border: 1px solid var(--line);
    background: var(--bg);
    color: var(--fg);
    font: inherit;
    width: 100%;
    box-sizing: border-box;
  }

  input:focus {
    outline: 2px solid var(--acc);
    border-color: transparent;
  }

  .form-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .btn-cancel {
    height: 36px;
    padding: 0 14px;
    border-radius: 18px;
    color: var(--mut);
    font-weight: 500;
  }

  .btn-submit {
    height: 36px;
    padding: 0 16px;
    border-radius: 18px;
    background: var(--acc);
    color: #ffffff;
    font-weight: 500;
    transition: opacity 0.15s;
  }

  .btn-submit:disabled {
    opacity: 0.4;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 64px 16px;
    color: var(--mut);
  }

  .empty-state p {
    font-size: 16px;
    margin: 0;
  }

  @media (max-width: 599px) {
    .albums-page { padding: 8px 8px 24px; }
    .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px 10px; }
    .create-pill { height: 38px; padding: 0 14px; }
    .add-photo { margin-left: 0; }
    .add-photo span { display: none; } /* tombol ikon saja agar judul album muat */
    .add-photo { width: 40px; padding: 0; justify-content: center; }
  }
</style>
