<script>
  import { onMount, untrack } from 'svelte';
  import Icon from '../../ui/Icon.svelte';
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte';
  import { createThumbState } from '../../ui/thumb.svelte.js';
  import ViewFab from './ViewFab.svelte';
  import SelectionBar from './SelectionBar.svelte';
  import TimelineTile from './TimelineTile.svelte';
  import TimelineListItem from './TimelineListItem.svelte';
  import { Selection } from './selection.svelte.js';
  import { dayLabel, groupItems, layoutRows, targetHeight } from './grouping.js';
  import { view } from '../../core/view.svelte.js';
  import { api } from '../../core/api.svelte.js';
  import { favorites, loadFavorites, toggleFavorite, toggleFavorites } from '../../core/favorites.svelte.js';
  import { showToast } from '../../core/toast.svelte.js';

  let {
    fetchUrl = (c) => api('/api/timeline?limit=100' + (c ? `&before=${c.before}&before_id=${c.before_id}` : '')),
    empty = 'Belum ada foto',
    searchQuery = '',
    filterFavorites = false,
    serverOrder = true, // false: sumber (mis. album) tak mendukung order=asc -> urutkan di klien
    photoId = null,
    onphotochange = () => {},
  } = $props();

  const GAP = 4;

  let items = $state([]);
  let done = $state(false);
  let ready = $state(false);
  let loadError = $state(false); // permintaan gagal (server tak terjangkau / ditolak)
  let width = $state(0);
  let openIndex = $state(-1);
  let picking = $state(false);
  let showBatchDeleteConfirm = $state(false);
  let batchDeleting = $state(false);

  const thumbs = createThumbState();
  const sel = new Selection((i) => openLightbox(i));

  let next = null;
  let loading = false;
  let sentinel;
  let controller;
  let destroyed = false;
  let Lightbox = $state(null);
  let AlbumPicker = $state(null);
  let routedPhotoId = null;

  // ---------- pemuatan data ----------

  async function load() {
    if (loading || done) return;
    loading = true;
    controller?.abort();
    controller = new AbortController();
    try {
      let url = fetchUrl(next);
      if (serverOrder && view.order === 'asc') url += (url.includes('?') ? '&' : '?') + 'order=asc';
      const response = await fetch(url, { signal: controller.signal });
      if (!response.ok) throw new Error(`timeline request failed: ${response.status}`);
      const data = await response.json();
      if (destroyed) return;
      // Deep link menyisipkan satu foto di awal; jangan duplikasi saat halamannya dimuat
      // (id ganda membuat {#each} ber-key melempar each_key_duplicate).
      const have = new Set(items.map((x) => x.id));
      items.push(...data.items.filter((x) => !have.has(x.id)));
      next = data.next ?? null;
      if (!data.next) done = true;
    } catch (err) {
      if (err.name !== 'AbortError') {
        done = true;
        loadError = true;
      }
    } finally {
      loading = false;
      ready = true;
      // IntersectionObserver hanya menembak saat visibilitas berubah. Bila sentinel masih terlihat
      // (mis. filter Favorit mengosongkan hasil halaman ini), lanjutkan memuat.
      requestAnimationFrame(() => {
        if (!destroyed && !done && sentinel && sentinel.getBoundingClientRect().top < window.innerHeight + 200) load();
      });
    }
  }

  function reload() {
    loadError = false;
    items = [];
    next = null;
    done = false;
    ready = false;
    load();
  }

  // Urutan berubah dari FAB -> muat ulang dari awal (urutan ditentukan server)
  let lastOrder = view.order;
  $effect(() => {
    const o = view.order;
    if (o === lastOrder) return;
    lastOrder = o;
    if (serverOrder) untrack(() => reload());
  });

  // ---------- turunan tampilan ----------

  let filteredItems = $derived.by(() => {
    let res = items;
    if (view.type !== 'all') res = res.filter((a) => a.type === view.type);
    if (!serverOrder && view.order === 'asc') res = [...res].reverse();
    if (filterFavorites) res = res.filter((a) => favorites.has(a.id));
    const q = searchQuery.trim().toLowerCase();
    if (q) {
      res = res.filter((a) => {
        if (q === 'video') return a.type === 'video';
        if (q === 'image' || q === 'foto') return a.type === 'image';
        if (q === 'hari ini') return dayLabel(a.taken_at).toLowerCase() === 'hari ini';
        return (a.path || '').toLowerCase().includes(q) || dayLabel(a.taken_at).toLowerCase().includes(q);
      });
    }
    return res;
  });

  let targetH = $derived(targetHeight(width, view.size));

  let groups = $derived(
    groupItems(filteredItems, view.group).map((g) => ({
      ...g,
      rows: layoutRows(g.list, view.layout, width, targetH, GAP),
      allSelected: g.list.length > 0 && g.list.every(({ a }) => sel.has(a.id)),
    }))
  );

  // ---------- Lightbox & deep link ----------

  async function openLightbox(index) {
    if (!Lightbox) Lightbox = (await import('../lightbox/Lightbox.svelte')).default;
    openIndex = index;
    const id = filteredItems[index]?.id;
    if (id) {
      routedPhotoId = id;
      onphotochange(id);
    }
  }

  async function openPhotoByID(id) {
    let index = items.findIndex((item) => item.id === id);
    if (index < 0) {
      try {
        const response = await fetch(api(`/api/assets/${id}`));
        if (!response.ok) throw new Error(`asset request failed: ${response.status}`);
        items.unshift(await response.json());
        index = 0;
      } catch {
        onphotochange(null);
        return;
      }
    }
    openLightbox(index);
  }

  function closeLightbox() {
    openIndex = -1;
    routedPhotoId = null;
    onphotochange(null);
  }

  function changeLightboxPhoto(id) {
    routedPhotoId = id;
    onphotochange(id);
  }

  $effect(() => {
    const id = Number(photoId);
    if (!id) {
      untrack(() => {
        if (routedPhotoId !== null) {
          openIndex = -1;
          routedPhotoId = null;
        }
      });
      return;
    }
    if (routedPhotoId !== id) {
      routedPhotoId = id;
      openPhotoByID(id);
    }
  });

  // ---------- aksi ----------

  async function handleBatchDelete() {
    if (sel.size === 0 || batchDeleting) return;
    batchDeleting = true;
    const ids = sel.list();
    try {
      const res = await fetch(api('/api/assets/delete'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids }),
      });
      if (!res.ok) throw new Error('Gagal menghapus foto.');
      const data = await res.json().catch(() => ({}));
      const idSet = new Set(ids);
      items = items.filter((a) => !idSet.has(a.id));
      sel.ids.clear();
      showBatchDeleteConfirm = false;
      showToast(`${data.deleted ?? ids.length} foto berhasil dihapus`, 2400);
    } catch (err) {
      alert(err.message || 'Gagal menghapus foto.');
    } finally {
      batchDeleting = false;
    }
  }

  function handleDeleteFromLightbox(id) {
    items = items.filter((a) => a.id !== id);
    if (openIndex >= filteredItems.length) openIndex = filteredItems.length - 1;
    if (filteredItems.length === 0) {
      closeLightbox();
    } else {
      const nextId = filteredItems[openIndex]?.id;
      if (nextId) changeLightboxPhoto(nextId);
      else closeLightbox();
    }
    showToast('1 foto berhasil dihapus', 2400);
  }

  async function openAlbumPicker() {
    if (!AlbumPicker) AlbumPicker = (await import('../albums/AlbumPicker.svelte')).default;
    picking = true;
  }

  function handleFavorite(id) {
    showToast(toggleFavorite(id) ? 'Ditambahkan ke favorit' : 'Dihapus dari favorit', 2000);
  }

  function handleBatchFavorite() {
    const n = sel.size;
    showToast(toggleFavorites(sel.list()) ? `${n} foto ditambahkan ke favorit` : `${n} foto dihapus dari favorit`, 2400);
  }

  function albumAdded(n, name) {
    picking = false;
    sel.ids.clear();
    showToast(`${n} item ditambahkan ke “${name}”`, 2800);
  }

  onMount(() => {
    loadFavorites();
    load();
    const io = new IntersectionObserver((es) => es[0].isIntersecting && load(), { rootMargin: '200px' });
    io.observe(sentinel);
    return () => {
      destroyed = true;
      controller?.abort();
      io.disconnect();
    };
  });
</script>

{#if sel.active}
  <SelectionBar
    count={sel.size}
    onclear={sel.exit}
    onalbum={openAlbumPicker}
    onfavorite={handleBatchFavorite}
    ondelete={() => (showBatchDeleteConfirm = true)}
  />
{/if}

<div class="wrap" class:selecting={sel.active} bind:clientWidth={width}>
  {#each groups as g (g.key)}
    <section class="date-group">
      {#if g.label}
        <div class="date-header">
          <div class="date-title-wrap">
            <!-- Kotak centang di kiri teks; tersembunyi (lebar 0) sampai teks tanggal di-hover -->
            <button
              class="day-check"
              class:checked={g.allSelected}
              onclick={() => sel.toggleGroup(g)}
              title="Pilih semua tanggal ini"
              aria-label="Pilih hari ini"
            >
              <Icon name={g.allSelected ? 'checkCircle' : 'circle'} size={20} />
            </button>
            <h2>{g.label}</h2>
          </div>
          <span class="day-count">{g.list.length} item</span>
        </div>
      {/if}

      {#if view.layout === 'list'}
        <div class="list">
          {#each g.list as { a, i } (a.id)}
            <TimelineListItem
              {a}
              {thumbs}
              selected={sel.has(a.id)}
              favorite={favorites.has(a.id)}
              onpress={() => sel.press(a.id)}
              onrelease={sel.release}
              ontap={() => sel.tap(i, a.id)}
              ontoggle={() => sel.toggle(a.id)}
            />
          {/each}
        </div>
      {:else}
        {#each g.rows as row, r (r)}
          <div class="row" style:height="{row.h}px">
            {#each row.tiles as { a, i, w } (a.id)}
              <TimelineTile
                {a}
                {w}
                {thumbs}
                fit={view.layout === 'square' ? 'cover' : 'contain'}
                selected={sel.has(a.id)}
                favorite={favorites.has(a.id)}
                onpress={() => sel.press(a.id)}
                onrelease={sel.release}
                ontap={() => sel.tap(i, a.id)}
                ontoggle={() => sel.toggle(a.id)}
              />
            {/each}
          </div>
        {/each}
      {/if}
    </section>
  {/each}

  <div bind:this={sentinel} class="end">
    {#if ready && done && loadError && items.length === 0}
      <div class="empty-state" role="alert">
        <Icon name="brokenImage" size={48} />
        <p>Gagal memuat foto dari server.</p>
        <button class="retry" onclick={reload}>Coba lagi</button>
      </div>
    {:else if ready && done && filteredItems.length === 0}
      <div class="empty-state">
        <Icon name="photo" size={48} />
        <p>{empty}</p>
      </div>
    {:else if !done}
      <div class="loading-state">
        <span>Memuat foto…</span>
      </div>
    {/if}
  </div>
</div>

{#if !sel.active && !(Lightbox && openIndex >= 0)}
  <ViewFab
    count={filteredItems.length}
    selectMode={sel.mode}
    ontoggleselect={() => (sel.mode = !sel.mode)}
    ontop={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
  />
{/if}

{#if Lightbox && openIndex >= 0 && filteredItems[openIndex]}
  <Lightbox
    items={filteredItems}
    bind:index={openIndex}
    onclose={closeLightbox}
    onchange={changeLightboxPhoto}
    ontogglefav={handleFavorite}
    ondelete={handleDeleteFromLightbox}
    {favorites}
  />
{/if}

{#if AlbumPicker && picking}
  <AlbumPicker ids={sel.list()} onclose={() => (picking = false)} ondone={albumAdded} />
{/if}

<ConfirmDialog
  open={showBatchDeleteConfirm}
  title={`Hapus ${sel.size} foto?`}
  message="Foto yang dipilih akan disembunyikan dari Pigura. File asli di disk tidak dihapus."
  confirmLabel="Hapus"
  cancelLabel="Batal"
  loading={batchDeleting}
  onconfirm={handleBatchDelete}
  oncancel={() => (showBatchDeleteConfirm = false)}
/>

<style>
  .wrap {
    padding: 8px 0 32px;
  }

  .date-group {
    margin-bottom: 24px;
  }

  .date-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 6px 8px 0; /* label rata dengan tepi kiri foto */
  }
  .date-title-wrap {
    display: flex;
    align-items: center;
  }
  h2 {
    font-size: 15px;
    font-weight: 500;
    color: var(--fg);
    letter-spacing: -0.2px;
  }
  .day-count {
    font-size: 12px;
    color: var(--mut);
  }

  /* Tersembunyi = lebar 0 (label rata kiri, tidak bergeser); membuka di kiri teks saat teks tanggal di-hover. */
  .day-check {
    flex: none;
    width: 0;
    height: 28px;
    margin-right: 0;
    border-radius: 50%;
    overflow: hidden;
    display: grid;
    place-items: center;
    color: var(--mut);
    opacity: 0;
    transition: width 0.15s, margin-right 0.15s, opacity 0.15s, color 0.15s;
  }
  .date-title-wrap:hover .day-check,
  .selecting .day-check,
  .day-check.checked {
    width: 28px;
    margin-right: 8px;
    opacity: 1;
  }
  .day-check.checked {
    color: var(--acc);
  }
  /* Layar sentuh: ruang checkbox tanggal tidak dipakai sampai mode pilih aktif (tekan lama) */
  @media (hover: none) {
    .wrap:not(.selecting) .day-check:not(.checked) { display: none; }
  }

  .row {
    display: flex;
    gap: 4px;
    margin-bottom: 4px;
  }
  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0 4px;
  }

  .end {
    text-align: center;
    color: var(--mut);
    padding: 48px 16px;
    min-height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--mut);
  }
  .empty-state p {
    font-size: 15px;
    margin: 0;
  }
  .retry {
    height: 38px;
    padding: 0 20px;
    border-radius: 19px;
    background: var(--acc-bg);
    color: var(--acc-fg);
    font-weight: 500;
    font-size: 14px;
  }

  @media (max-width: 599px) {
    :global(.vfab) { right: 14px; }
  }
</style>
