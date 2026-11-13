<script>
  import { onMount } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import Icon from '../../ui/Icon.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { createThumbState } from '../../ui/thumb.svelte.js';
  import { showToast } from '../../core/toast.svelte.js';
  import { fmtSize, dateFmt, baseName } from '../../core/format.js';
  import { api } from '../../core/api.svelte.js';
  import { link } from '../../core/router.svelte.js';

  const PAGE = 20;

  let status = $state({ running: false, phase: '', done: 0, total: 0, finished_at: 0, error: '' });
  let summary = $state({ groups: 0, extra_files: 0, wasted_bytes: 0 });
  let groups = $state([]);
  let loaded = $state(false);
  let more = $state(false);
  let busy = $state(false);
  let removing = $state(false);
  let timer;
  let wasRunning = false;

  const marked = new SvelteSet(); // id yang akan dihapus dari galeri
  const seen = new SvelteSet(); // grup yang sudah diberi pilihan default
  const thumbs = createThumbState();

  const name = baseName;
  const dir = (p) => p.split('/').slice(0, -1).join('/');

  function applyDefaults(list) {
    for (const g of list) {
      if (seen.has(g.hash)) continue;
      seen.add(g.hash);
      g.items.slice(1).forEach((a) => marked.add(a.id)); // item tertua dipertahankan
    }
  }

  async function fetchPage(offset, limit = PAGE) {
    const res = await fetch(api(`/api/duplicates?limit=${limit}&offset=${offset}`));
    if (!res.ok) throw new Error(String(res.status));
    return res.json();
  }

  async function refresh() {
    try {
      const d = await fetchPage(0);
      status = d.status;
      summary = d.summary;
      groups = d.groups;
      more = d.groups.length < d.summary.groups;
      applyDefaults(d.groups);
      // tampilan berubah saat pemindaian baru selesai
      if (wasRunning && !d.status.running) flash('Pemeriksaan selesai');
      wasRunning = d.status.running;
    } catch {
      /* server sibuk/offline */
    } finally {
      loaded = true;
      clearTimeout(timer);
      if (status.running) timer = setTimeout(poll, 1000);
    }
  }

  // saat berjalan, hanya ambil status ringan (limit=1) lalu muat penuh ketika selesai
  async function poll() {
    try {
      const d = await fetchPage(0, 1);
      status = d.status;
      if (!d.status.running) return refresh();
    } catch {
      /* abaikan */
    }
    timer = setTimeout(poll, 1000);
  }

  async function loadMore() {
    try {
      const d = await fetchPage(groups.length);
      groups.push(...d.groups);
      more = groups.length < d.summary.groups && d.groups.length > 0;
      applyDefaults(d.groups);
    } catch {
      /* ignore */
    }
  }

  async function start() {
    if (busy || status.running) return;
    busy = true;
    try {
      await fetch(api('/api/duplicates/scan'), { method: 'POST' });
      wasRunning = true;
      status = { ...status, running: true, phase: 'sample', done: 0, total: 0, error: '' };
      clearTimeout(timer);
      timer = setTimeout(poll, 500);
    } finally {
      busy = false;
    }
  }

  const flash = (msg) => showToast(msg, 3000, 62); // naik di atas dock aksi

  // minimal satu salinan di tiap grup harus tetap tidak ditandai
  function toggle(g, id) {
    if (marked.has(id)) {
      marked.delete(id);
    } else {
      const unmarked = g.items.filter((a) => !marked.has(a.id)).length;
      if (unmarked <= 1) return flash('Sisakan minimal satu salinan di tiap grup');
      marked.add(id);
    }
  }

  function keepOnly(g, id) {
    g.items.forEach((a) => (a.id === id ? marked.delete(a.id) : marked.add(a.id)));
  }

  let markedList = $derived(
    groups.flatMap((g) => g.items.filter((a) => marked.has(a.id)))
  );
  let markedBytes = $derived(markedList.reduce((n, a) => n + a.size, 0));

  async function removeMarked() {
    if (!markedList.length || removing) return;
    if (
      !confirm(
        `Hapus ${markedList.length} salinan (${fmtSize(markedBytes)}) dari galeri?\n\nFile di disk tidak disentuh; foto hanya disembunyikan dari galeri.`
      )
    )
      return;
    removing = true;
    try {
      const ids = markedList.map((a) => a.id);
      for (let i = 0; i < ids.length; i += 500) {
        const res = await fetch(api('/api/assets/delete'), {
          method: 'POST',
          body: JSON.stringify({ ids: ids.slice(i, i + 500) }),
        });
        if (!res.ok) throw new Error(String(res.status));
      }
      ids.forEach((id) => marked.delete(id));
      flash(`${ids.length} salinan dihapus dari galeri`);
      await refresh();
    } catch {
      flash('Gagal menghapus, coba lagi');
    } finally {
      removing = false;
    }
  }

  let pct = $derived(status.total ? Math.round((status.done / status.total) * 100) : 0);
  let phaseLabel = $derived(
    status.phase === 'full' ? 'Memverifikasi isi file' : 'Memeriksa kandidat (sampel file)'
  );

  onMount(() => {
    refresh();
    return () => {
      clearTimeout(timer);
      clearTimeout(toastTimer);
    };
  });
</script>

<div class="dup">
  <div class="bar">
    <div>
      <h1>Cek duplikat</h1>
      <p class="lead">
        Mencari file dengan isi identik. Hanya file berukuran sama yang dibaca, dan hasilnya disimpan
        sehingga pemeriksaan berikutnya cepat.
      </p>
    </div>
    <button class="pill" onclick={start} disabled={busy || status.running}>
      <Icon name="refresh" size={18} class={status.running ? 'spin' : ''} />
      <span>{status.running ? 'Memeriksa…' : status.finished_at ? 'Periksa ulang' : 'Cari duplikat'}</span>
    </button>
  </div>

  {#if status.running}
    <div class="prog" role="status" aria-live="polite">
      <div class="prog-top">
        <span>{phaseLabel}</span>
        <span>{status.total ? `${status.done}/${status.total}` : 'Menyiapkan…'}</span>
      </div>
      <div class="track"><div class="fill" style:width="{pct}%"></div></div>
    </div>
  {:else if status.error}
    <p class="err" role="alert">Pemeriksaan gagal: {status.error}</p>
  {/if}

  {#if loaded && !status.running}
    {#if summary.groups > 0}
      <div class="sum">
        <div class="stat"><b>{summary.groups}</b><span>grup duplikat</span></div>
        <div class="stat"><b>{summary.extra_files}</b><span>salinan berlebih</span></div>
        <div class="stat"><b>{fmtSize(summary.wasted_bytes)}</b><span>dapat dihemat</span></div>
      </div>
    {:else}
      <div class="empty">
        <Icon name="copy" size={48} />
        <p>
          {status.finished_at ? 'Tidak ada duplikat yang ditemukan' : 'Belum ada hasil. Jalankan pemeriksaan untuk memulai.'}
        </p>
      </div>
    {/if}
  {/if}

  {#each groups as g (g.hash + g.size)}
    <section class="group">
      <header>
        <span class="g-title">{g.count} salinan · {fmtSize(g.size)}</span>
        <span class="g-save">hemat {fmtSize(g.size * (g.count - 1))}</span>
      </header>
      <div class="items">
        {#each g.items as a, i (a.id)}
          {@const isMarked = marked.has(a.id)}
          <div class="item" class:marked={isMarked}>
            <a class="thumb" href={`/photos/${a.id}`} onclick={link} aria-label="Buka {name(a.path)}">
              <Thumb src={api(`/api/assets/${a.id}/thumb`)} id={a.id} state={thumbs} />
              {#if a.type === 'video'}<span class="vid"><Icon name="play" size={12} /></span>{/if}
            </a>
            <div class="meta">
              <div class="fn" title={a.path}>{name(a.path)}</div>
              <div class="sub" title={dir(a.path)}>{dir(a.path) || '/'}</div>
              <div class="sub">{dateFmt.day.format(new Date(a.taken_at * 1000))}</div>
              <div class="acts">
                {#if i === 0}<span class="badge">Tertua</span>{/if}
                <button class="chip" class:on={!isMarked} onclick={() => keepOnly(g, a.id)} title="Simpan ini, hapus lainnya">
                  Simpan ini
                </button>
                <button class="chip del" class:on={isMarked} onclick={() => toggle(g, a.id)} aria-pressed={isMarked}>
                  {isMarked ? 'Akan dihapus' : 'Hapus'}
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/each}

  {#if more}
    <button class="more" onclick={loadMore}>Muat grup lainnya</button>
  {/if}

  {#if markedList.length > 0}
    <div class="dock">
      <span>{markedList.length} dipilih · {fmtSize(markedBytes)}</span>
      <button class="danger" onclick={removeMarked} disabled={removing}>
        <Icon name="trash" size={18} />
        <span>{removing ? 'Menghapus…' : 'Hapus dari galeri'}</span>
      </button>
    </div>
  {/if}

</div>

<style>
  .dup { padding: 16px 12px 120px; max-width: 960px; }
  .bar { display: flex; flex-wrap: wrap; gap: 12px 16px; align-items: flex-start; justify-content: space-between; margin-bottom: 20px; padding: 0 4px; }
  h1 { font-size: 22px; font-weight: 500; letter-spacing: -0.3px; color: var(--fg); margin: 0 0 4px; }
  .lead { margin: 0; font-size: 13px; color: var(--mut); max-width: 520px; line-height: 1.45; }
  .pill {
    display: inline-flex; align-items: center; gap: 8px; height: 40px; padding: 0 18px; border-radius: 20px;
    background: var(--acc-bg); color: var(--acc-fg); font-size: 14px; font-weight: 500;
  }
  .pill:disabled { opacity: 0.6; }

  .prog { margin: 0 4px 20px; padding: 14px 16px; border-radius: 14px; background: var(--surface-variant); }
  .prog-top { display: flex; justify-content: space-between; font-size: 13px; color: var(--fg-variant); margin-bottom: 10px; }
  .track { height: 6px; border-radius: 3px; background: var(--line); overflow: hidden; }
  .fill { height: 100%; background: var(--acc); border-radius: 3px; transition: width 0.3s; }
  .err { margin: 0 4px 16px; color: #d93025; font-size: 13px; }

  .sum { display: flex; flex-wrap: wrap; gap: 10px; margin: 0 4px 20px; }
  .stat { flex: 1 1 140px; padding: 14px 16px; border-radius: 14px; background: var(--surface-variant); display: flex; flex-direction: column; gap: 2px; }
  .stat b { font-size: 20px; font-weight: 600; color: var(--fg); }
  .stat span { font-size: 12px; color: var(--mut); }

  .empty { display: flex; flex-direction: column; align-items: center; gap: 12px; padding: 56px 16px; color: var(--mut); text-align: center; }
  .empty p { margin: 0; font-size: 15px; }

  .group { margin-bottom: 16px; border: 1px solid var(--line); border-radius: 16px; overflow: hidden; background: var(--surface); }
  .group header { display: flex; justify-content: space-between; gap: 8px; padding: 10px 14px; background: var(--surface-variant); font-size: 13px; }
  .g-title { font-weight: 500; color: var(--fg); }
  .g-save { color: var(--mut); }
  .items { display: flex; flex-direction: column; }
  .item { display: flex; gap: 12px; padding: 10px 14px; border-top: 1px solid var(--line-subtle); transition: background 0.15s; }
  .item:first-child { border-top: 0; }
  .item.marked { background: rgba(217, 48, 37, 0.07); }

  .thumb {
    position: relative; flex: none; width: 84px; height: 84px; border-radius: 10px; overflow: hidden;
    background: var(--tile); display: block;
  }
  .item.marked .thumb :global(img) { filter: grayscale(0.6); }
  .vid { position: absolute; right: 5px; bottom: 5px; display: grid; place-items: center; width: 20px; height: 20px; border-radius: 50%; background: rgba(0, 0, 0, 0.6); color: #fff; }

  .meta { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 2px; }
  .fn { font-size: 14px; font-weight: 500; color: var(--fg); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .sub { font-size: 12px; color: var(--mut); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .acts { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: 6px; }
  .badge { font-size: 11px; padding: 2px 8px; border-radius: 10px; background: var(--acc-bg); color: var(--acc-fg); font-weight: 500; }
  .chip { height: 28px; padding: 0 12px; border-radius: 14px; border: 1px solid var(--line); font-size: 12px; font-weight: 500; color: var(--fg-variant); }
  .chip:hover { background: var(--hover); }
  .chip.on:not(.del) { background: var(--acc-bg); color: var(--acc-fg); border-color: var(--acc); }
  .chip.del.on { background: #d93025; border-color: #d93025; color: #fff; }

  .more { display: block; margin: 8px auto 0; height: 38px; padding: 0 20px; border-radius: 19px; border: 1px solid var(--line); font-size: 14px; color: var(--fg); }
  .more:hover { background: var(--hover); }

  .dock {
    position: fixed; left: 50%; transform: translateX(-50%); z-index: 40;
    bottom: calc(84px + env(safe-area-inset-bottom));
    display: flex; align-items: center; gap: 16px; padding: 10px 12px 10px 20px; border-radius: 28px;
    background: var(--fg); color: var(--bg); font-size: 14px; box-shadow: 0 6px 24px rgba(0, 0, 0, 0.3);
    max-width: calc(100vw - 24px);
  }
  @media (min-width: 900px) { .dock { bottom: 24px; } }
  .danger { display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px; border-radius: 19px; background: #d93025; color: #fff; font-weight: 500; font-size: 14px; }
  .danger:disabled { opacity: 0.6; }


  @media (pointer: coarse) {
    .chip { height: 34px; }
  }
  @media (max-width: 480px) {
    .dup { padding-left: 8px; padding-right: 8px; }
    .thumb { width: 72px; height: 72px; }
    .bar .pill { width: 100%; justify-content: center; }
  }
</style>
