<script>
  import Icon from '../../ui/Icon.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { api } from '../../core/api.svelte.js';
  import { fmtSize, formatDuration, dateFmt, baseName } from '../../core/format.js';

  // Satu baris foto/video pada layout daftar.
  let { a, selected, favorite, thumbs, onpress, onrelease, ontap, ontoggle } = $props();
</script>

<div class="li" class:sel={selected} role="group" aria-label={a.path || 'Foto'} oncontextmenu={(e) => e.preventDefault()}>
  <button class="main" onpointerdown={onpress} onpointerup={onrelease} onpointerleave={onrelease} onclick={ontap}>
    <span class="thumb">
      <Thumb src={api(`/api/assets/${a.id}/thumb${a.taken_at ? `?t=${a.taken_at}` : ''}`)} id={a.id} state={thumbs} />
      {#if a.type === 'video'}<span class="play"><Icon name="play" size={12} /></span>{/if}
    </span>
    <span class="meta">
      <span class="name">{baseName(a.path) || 'Foto'}</span>
      <span class="sub">
        {dateFmt.dayTime.format(new Date(a.taken_at * 1000))} · {fmtSize(a.size)}{a.type === 'video' && a.duration > 0 ? ` · ${formatDuration(a.duration)}` : ''}
      </span>
    </span>
  </button>
  {#if favorite}<span class="star"><Icon name="starFilled" size={16} /></span>{/if}
  <button class="check" class:selected aria-label="Pilih foto" onclick={ontoggle}>
    <Icon name={selected ? 'checkCircle' : 'circle'} size={22} />
  </button>
</div>

<style>
  .li { position: relative; display: flex; align-items: center; border-radius: 12px; transition: background 0.15s; }
  .li:hover { background: var(--hover-subtle); }
  .li.sel { background: var(--acc-bg); }
  .main { flex: 1; min-width: 0; justify-content: flex-start; gap: 12px; padding: 6px 8px; text-align: left; }
  .thumb { position: relative; flex: none; width: 56px; height: 56px; border-radius: 10px; overflow: hidden; display: block; }
  .play {
    position: absolute; right: 3px; bottom: 3px; width: 18px; height: 18px; border-radius: 50%;
    display: grid; place-items: center; background: rgba(0, 0, 0, 0.6); color: #fff;
  }
  .meta { display: flex; flex-direction: column; min-width: 0; gap: 1px; }
  .name { font-size: 14px; font-weight: 500; color: var(--fg); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .sub { font-size: 12px; color: var(--mut); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .star { color: #fbbc04; display: grid; padding: 0 2px; }
  .check { width: 44px; height: 44px; border-radius: 50%; color: var(--mut); flex: none; }
  .check.selected { color: var(--acc); }
</style>
