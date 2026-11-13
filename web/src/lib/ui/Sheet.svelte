<script>
  import Icon from './Icon.svelte';

  // Modal bawah (bottom sheet di ponsel, kartu di tengah pada layar lebar).
  //   title   : bila ada, header standar (judul + tombol tutup); atau beri snippet `header`
  //   width   : lebar maksimum panel;  height: tinggi tetap (kosong = menyesuaikan isi, maks maxHeight)
  let {
    onclose,
    label,
    title = '',
    header = undefined,
    children,
    width = '480px',
    height = '',
    maxHeight = '80dvh',
    radius = '16px',
    padding = '16px 8px',
    padBottom = '16px',
    gap = '0',
    z = 60,
    dim = 0.5,
    scroll = true,
  } = $props();
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="scrim" role="presentation" style:z-index={z} style:--dim={dim} onclick={onclose}>
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div
    class="sheet"
    class:scroll
    role="dialog"
    aria-modal="true"
    aria-label={label ?? title}
    tabindex="-1"
    style:--w={width}
    style:--h={height || 'auto'}
    style:--mh={maxHeight}
    style:--r={radius}
    style:--pad={padding}
    style:--pad-b={padBottom}
    style:--gap={gap}
    onclick={(e) => e.stopPropagation()}
  >
    {#if header}
      {@render header()}
    {:else if title}
      <header>
        <h3>{title}</h3>
        <button class="close" onclick={onclose} aria-label="Tutup"><Icon name="close" size={20} /></button>
      </header>
    {/if}
    {@render children()}
  </div>
</div>

<style>
  .scrim {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, var(--dim));
    display: grid;
    place-items: end center;
  }
  .sheet {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    width: min(var(--w), 100%);
    height: var(--h);
    max-height: var(--mh);
    background: var(--bg);
    color: var(--fg);
    border-radius: var(--r) var(--r) 0 0;
    padding: var(--pad);
    padding-bottom: calc(var(--pad-b) + env(safe-area-inset-bottom));
    overflow: hidden;
  }
  .sheet.scroll { overflow-y: auto; }
  @media (min-width: 600px) {
    .scrim { place-items: center; }
    .sheet { border-radius: var(--r); }
  }
  header { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
  h3 { font-size: 17px; font-weight: 600; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin: 0; }
  .close { width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; color: var(--mut); flex: none; }
  .close:hover { background: var(--hover); color: var(--fg); }
</style>
