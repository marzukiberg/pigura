<script>
  import { Popover } from 'bits-ui';
  import Icon from '../../ui/Icon.svelte';
  import Seg from '../../ui/Seg.svelte';
  import { view, setView, resetView, isCustomView } from '../../core/view.svelte.js';

  // count: jumlah foto tampil; selectMode/ontoggleselect: mode pilih; ontop: gulir ke atas
  let { count = 0, selectMode = false, ontoggleselect, ontop } = $props();
  let open = $state(false);
  let custom = $derived(isCustomView());

  const LAYOUTS = [
    { id: 'justified', label: 'Grid', icon: 'grid' },
    { id: 'square', label: 'Kotak', icon: 'square' },
    { id: 'list', label: 'Daftar', icon: 'list' },
  ];
  const SIZES = [
    { id: 'sm', label: 'Kecil' },
    { id: 'md', label: 'Sedang' },
    { id: 'lg', label: 'Besar' },
  ];
  const TYPES = [
    { id: 'all', label: 'Semua' },
    { id: 'image', label: 'Foto' },
    { id: 'video', label: 'Video' },
  ];
  const ORDERS = [
    { id: 'desc', label: 'Terbaru' },
    { id: 'asc', label: 'Terlama' },
  ];
  const GROUPS = [
    { id: 'day', label: 'Hari' },
    { id: 'month', label: 'Bulan' },
    { id: 'none', label: 'Tanpa' },
  ];
</script>

<Popover.Root bind:open>
  <Popover.Trigger class="vfab" aria-label="Tampilan dan filter" title="Tampilan & filter">
    <Icon name="sliders" size={24} />
    {#if custom}<span class="vfab-dot" aria-hidden="true"></span>{/if}
  </Popover.Trigger>
  <Popover.Portal>
    <Popover.Content class="vfab-pop" side="top" align="end" sideOffset={12} collisionPadding={12}>
      <div class="vp-head">
        <h3>Tampilan</h3>
        <span class="vp-count">{count} item</span>
      </div>

      <div class="vp-sec">
        <div class="vp-lbl">Layout</div>
        <Seg options={LAYOUTS} value={view.layout} onchange={(v) => setView('layout', v)} label="Layout" />
      </div>

      {#if view.layout !== 'list'}
        <div class="vp-sec">
          <div class="vp-lbl">Ukuran</div>
          <Seg options={SIZES} value={view.size} onchange={(v) => setView('size', v)} label="Ukuran" />
        </div>
      {/if}

      <div class="vp-sec">
        <div class="vp-lbl">Jenis</div>
        <Seg options={TYPES} value={view.type} onchange={(v) => setView('type', v)} label="Jenis" />
      </div>

      <div class="vp-sec">
        <div class="vp-lbl">Urutan</div>
        <Seg options={ORDERS} value={view.order} onchange={(v) => setView('order', v)} label="Urutan" />
      </div>

      <div class="vp-sec">
        <div class="vp-lbl">Kelompokkan per</div>
        <Seg options={GROUPS} value={view.group} onchange={(v) => setView('group', v)} label="Kelompok" />
      </div>

      <div class="vp-acts">
        <button
          onclick={() => {
            ontoggleselect?.();
            open = false;
          }}
        >
          <Icon name="checkSquare" size={18} /><span>{selectMode ? 'Batal pilih' : 'Pilih foto'}</span>
        </button>
        <button
          onclick={() => {
            ontop?.();
            open = false;
          }}
        >
          <Icon name="up" size={18} /><span>Ke atas</span>
        </button>
        <button onclick={resetView} disabled={!custom}>
          <Icon name="refresh" size={18} /><span>Reset</span>
        </button>
      </div>
    </Popover.Content>
  </Popover.Portal>
</Popover.Root>

<style>
  :global(.vfab) {
    position: fixed;
    z-index: 35;
    right: 16px;
    bottom: calc(80px + env(safe-area-inset-bottom));
    width: 56px;
    height: 56px;
    border-radius: 18px;
    background: var(--acc-bg);
    color: var(--acc-fg);
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35);
    transition: transform 0.15s, box-shadow 0.15s;
  }
  :global(.vfab:active) { transform: scale(0.94); }
  :global(.vfab[data-state='open']) { filter: brightness(1.1); }
  @media (min-width: 900px) {
    :global(.vfab) { right: 28px; bottom: 28px; }
  }
  .vfab-dot {
    position: absolute;
    top: 9px;
    right: 9px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #fbbc04;
    border: 2px solid var(--acc-bg);
  }

  :global(.vfab-pop) {
    z-index: 60;
    width: min(340px, calc(100vw - 24px));
    max-height: min(78dvh, 640px);
    overflow-y: auto;
    padding: 14px;
    border-radius: 20px;
    background: var(--surface);
    color: var(--fg);
    border: 1px solid var(--line);
    box-shadow: 0 8px 28px rgba(0, 0, 0, 0.3);
    outline: none;
  }
  .vp-head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 10px; }
  .vp-head h3 { font-size: 16px; font-weight: 600; }
  .vp-count { font-size: 12px; color: var(--mut); }
  .vp-sec { margin-bottom: 12px; }
  .vp-lbl { font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--mut); margin-bottom: 6px; }

  .vp-acts { display: flex; gap: 6px; padding-top: 10px; border-top: 1px solid var(--line); }
  .vp-acts button {
    flex: 1;
    flex-direction: column;
    gap: 4px;
    height: 56px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
    color: var(--fg-variant);
  }
  .vp-acts button:hover:not(:disabled) { background: var(--hover); }
  .vp-acts button:disabled { opacity: 0.4; cursor: default; }
</style>
