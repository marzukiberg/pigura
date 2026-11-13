<script>
  import Icon from './Icon.svelte';

  // Gambar thumbnail dengan shimmer saat memuat, fade-in setelah selesai, dan placeholder bila gagal.
  // state: createThumbState() milik daftar (agar status bertahan antar render ulang); id: kunci aset.
  let { src, id, state, fit = 'cover', fallback = undefined, alt = '' } = $props();

  const loaded = $derived(state.loaded.has(id));
  const failed = $derived(state.failed.has(id));
</script>

<span class="thumb" class:shimmer={!loaded && !failed}>
  {#if failed}
    {#if fallback}
      {@render fallback()}
    {:else}
      <span class="fb"><Icon name="brokenImage" size={24} /></span>
    {/if}
  {:else}
    <img
      {src}
      {alt}
      class:loaded
      style:object-fit={fit}
      loading="lazy"
      decoding="async"
      draggable="false"
      onload={() => state.loaded.add(id)}
      onerror={() => state.failed.add(id)}
    />
  {/if}
</span>

<style>
  .thumb {
    display: block;
    position: relative;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background: var(--thumb-bg, var(--tile));
  }
  .thumb:global(.shimmer) {
    background: var(--tile-shimmer);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite ease-in-out;
  }
  img {
    display: block;
    width: 100%;
    height: 100%;
    opacity: 0;
    transition: opacity 0.2s ease-out;
  }
  img.loaded {
    opacity: 1;
  }
  .fb {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    color: var(--mut);
  }
</style>
