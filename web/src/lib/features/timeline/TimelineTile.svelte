<script>
  import Icon from '../../ui/Icon.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { api } from '../../core/api.svelte.js';
  import { formatDuration, baseName } from '../../core/format.js';

  // Satu tile foto/video pada grid (layout rata atau kotak).
  let { a, w, fit = 'contain', selected, favorite, thumbs, onpress, onrelease, ontap, ontoggle } = $props();
</script>

<div
  class="tile"
  class:sel={selected}
  style:width="{w}px"
  role="group"
  aria-label={a.path || 'Foto'}
  oncontextmenu={(e) => e.preventDefault()}
>
  <button
    class="img-btn"
    aria-label="Buka foto"
    onpointerdown={onpress}
    onpointerup={onrelease}
    onpointerleave={onrelease}
    onclick={ontap}
  >
    <Thumb src={api(`/api/assets/${a.id}/thumb${a.taken_at ? `?t=${a.taken_at}` : ''}`)} id={a.id} state={thumbs} {fit}>
      {#snippet fallback()}
        <!-- Placeholder bila thumbnail gagal dimuat -->
        <div class="thumb-fallback">
          <Icon name="brokenImage" size={32} />
          <span class="fallback-name">{baseName(a.path) || 'Foto'}</span>
        </div>
      {/snippet}
    </Thumb>
  </button>

  <!-- Kotak centang (kiri atas) -->
  <button
    class="check-btn"
    class:selected
    aria-label="Pilih foto"
    onclick={(e) => {
      e.stopPropagation();
      ontoggle();
    }}
  >
    <Icon name={selected ? 'checkCircle' : 'circle'} size={22} />
  </button>

  {#if favorite}
    <span class="star-badge" title="Favorit"><Icon name="starFilled" size={16} /></span>
  {/if}

  {#if a.type === 'video'}
    <span class="vbadge">
      <Icon name="play" size={14} />
      {#if a.duration > 0}<span>{formatDuration(a.duration)}</span>{/if}
    </span>
  {/if}
</div>

<style>
  .tile {
    position: relative;
    flex: none;
    height: 100%;
    background: var(--tile);
    overflow: hidden;
    user-select: none;
    -webkit-touch-callout: none;
    transition: transform 0.15s ease-out, border-radius 0.15s ease-out;
  }
  .tile.sel {
    border-radius: 12px;
    outline: 3px solid var(--acc);
    outline-offset: -3px;
  }
  .tile.sel :global(img) {
    transform: scale(0.88);
    border-radius: 8px;
  }

  .img-btn {
    display: block;
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
  }

  .thumb-fallback {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    background: var(--surface-variant);
    color: var(--mut);
    padding: 8px;
    text-align: center;
  }
  .fallback-name {
    font-size: 11px;
    max-width: 90%;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Gradien atas saat hover/mode pilih (agar kotak centang terbaca) */
  .tile::after {
    content: '';
    position: absolute;
    inset: 0 0 auto 0;
    height: 56px;
    pointer-events: none;
    opacity: 0;
    background: linear-gradient(rgba(0, 0, 0, 0.4), transparent);
    transition: opacity 0.15s;
    z-index: 2;
  }
  .tile:hover::after,
  :global(.selecting) .tile::after {
    opacity: 1;
  }

  .check-btn {
    position: absolute;
    top: 6px;
    left: 6px;
    z-index: 5;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    color: rgba(255, 255, 255, 0.9);
    opacity: 0;
    transition: opacity 0.15s, transform 0.15s, color 0.15s;
    display: grid;
    place-items: center;
  }
  .tile:hover .check-btn,
  :global(.selecting) .check-btn {
    opacity: 1;
  }
  .check-btn.selected {
    opacity: 1;
    color: var(--acc);
    background: #ffffff;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  }

  .star-badge {
    position: absolute;
    bottom: 6px;
    left: 6px;
    z-index: 4;
    color: #fbbc04;
    display: grid;
    place-items: center;
    background: rgba(0, 0, 0, 0.45);
    border-radius: 50%;
    padding: 4px;
  }

  .vbadge {
    position: absolute;
    right: 6px;
    top: 6px;
    z-index: 4;
    color: #ffffff;
    display: flex;
    align-items: center;
    gap: 4px;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
    border-radius: 12px;
    padding: 2px 8px;
    font-size: 11px;
    font-weight: 500;
  }
</style>
