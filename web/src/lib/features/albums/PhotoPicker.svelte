<script>
  import { onMount } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import Icon from '../../ui/Icon.svelte';
  import Sheet from '../../ui/Sheet.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { createThumbState } from '../../ui/thumb.svelte.js';
  import { api } from '../../core/api.svelte.js';

  // Modal pilih foto dari library untuk ditambahkan ke sebuah album.
  let { albumId, albumName = '', onclose, ondone } = $props();

  let items = $state([]);
  let done = $state(false);
  let ready = $state(false);
  let busy = $state(false);
  let error = $state('');
  let sentinel;
  let next = null;
  let loading = false;

  const selected = new SvelteSet();
  const existing = new SvelteSet();
  const thumbs = createThumbState();

  async function load() {
    if (loading || done) return;
    loading = true;
    try {
      const res = await fetch(
        api('/api/timeline?limit=120' + (next ? `&before=${next.before}&before_id=${next.before_id}` : ''))
      );
      if (!res.ok) throw new Error(String(res.status));
      const data = await res.json();
      items.push(...data.items.filter((a) => a.type === 'image' || a.type === 'video'));
      next = data.next ?? null;
      if (!data.next) done = true;
    } catch {
      error = 'Foto gagal dimuat.';
      done = true;
    } finally {
      loading = false;
      ready = true;
    }
  }

  function toggle(a) {
    if (existing.has(a.id)) return;
    if (selected.has(a.id)) selected.delete(a.id);
    else selected.add(a.id);
  }

  async function add() {
    if (!selected.size || busy) return;
    busy = true;
    error = '';
    try {
      const res = await fetch(api(`/api/albums/${albumId}/assets`), {
        method: 'POST',
        body: JSON.stringify({ ids: [...selected] }),
      });
      const out = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(out.error || 'Gagal menambahkan foto.');
      ondone(out.added ?? selected.size);
    } catch (e) {
      error = e.message || 'Gagal menambahkan foto.';
    } finally {
      busy = false;
    }
  }

  onMount(() => {
    // tandai foto yang sudah ada di album
    fetch(api(`/api/albums/${albumId}/assets`))
      .then((r) => r.json())
      .then((d) => (d.items ?? []).forEach((a) => existing.add(a.id)))
      .catch(() => {});
    load();
    const io = new IntersectionObserver((es) => es[0].isIntersecting && load(), { rootMargin: '300px' });
    io.observe(sentinel);
    return () => io.disconnect();
  });
</script>

<Sheet {onclose} label="Tambah foto ke album" z={70} width="720px" height="min(86dvh, 760px)" maxHeight="none" padding="0" padBottom="0" scroll={false}>
  {#snippet header()}
    <header>
      <button class="ib" onclick={onclose} aria-label="Tutup"><Icon name="close" size={20} /></button>
      <div class="ttl">
        <h3>Tambah foto</h3>
        <small>{selected.size ? `${selected.size} dipilih` : `ke “${albumName}”`}</small>
      </div>
      <button class="go" disabled={!selected.size || busy} onclick={add}>
        {busy ? 'Menambahkan…' : 'Tambahkan'}
      </button>
    </header>
  {/snippet}

    {#if error}<p class="err" role="alert">{error}</p>{/if}

    <div class="scroll">
    <div class="grid">
      {#each items as a (a.id)}
        {@const isExisting = existing.has(a.id)}
        <button
          class="cell"
          class:sel={selected.has(a.id)}
          class:exists={isExisting}
          onclick={() => toggle(a)}
          aria-pressed={selected.has(a.id)}
          aria-label={isExisting ? 'Sudah ada di album' : 'Pilih foto'}
          disabled={isExisting}
        >
          <Thumb src={api(`/api/assets/${a.id}/thumb`)} id={a.id} state={thumbs} />
          {#if a.type === 'video'}<span class="vid"><Icon name="play" size={12} /></span>{/if}
          <span class="mark">
            <Icon name={selected.has(a.id) || isExisting ? 'checkCircle' : 'circle'} size={22} />
          </span>
        </button>
      {/each}
    </div>

    <div bind:this={sentinel} class="end">
      {#if !ready || (!done && loading)}
        <span>Memuat…</span>
      {:else if ready && items.length === 0}
        <span>Belum ada foto di library.</span>
      {/if}
    </div>
    </div>
</Sheet>

<style>
  header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--line);
    flex: none;
  }
  .ttl { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  h3 { font-size: 16px; font-weight: 500; margin: 0; }
  small { color: var(--mut); font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ib {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: var(--mut);
  }
  .ib:hover { background: var(--hover); color: var(--fg); }
  .go {
    height: 36px;
    padding: 0 18px;
    border-radius: 18px;
    background: var(--acc);
    color: #fff;
    font-weight: 500;
    font-size: 14px;
  }
  .go:disabled { opacity: 0.4; }

  .err { margin: 8px 16px 0; color: #d93025; font-size: 13px; }

  .scroll { flex: 1; overflow-y: auto; min-height: 0; }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
    gap: 3px;
    padding: 3px;
    align-content: start;
  }
  .cell {
    position: relative;
    aspect-ratio: 1;
    overflow: hidden;
    background: var(--tile);
    padding: 0;
  }
  .cell.sel { background: var(--acc-bg); --thumb-bg: transparent; }
  .cell.sel :global(img) { transform: scale(0.86); border-radius: 8px; }
  .cell.exists :global(img) { opacity: 0.35; }
  .cell:disabled { cursor: default; }
  .vid {
    position: absolute;
    right: 6px;
    bottom: 6px;
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.6);
    color: #fff;
  }
  .mark {
    position: absolute;
    top: 6px;
    left: 6px;
    display: grid;
    color: #fff;
    filter: drop-shadow(0 0 2px rgba(0, 0, 0, 0.6));
  }
  .cell.sel .mark, .cell.exists .mark { color: var(--acc); filter: none; background: #fff; border-radius: 50%; }

  .end { flex: none; min-height: 44px; display: grid; place-items: center; color: var(--mut); font-size: 13px; }
</style>
