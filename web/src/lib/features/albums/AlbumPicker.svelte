<script>
  import { onMount } from 'svelte';
  import Icon from '../../ui/Icon.svelte';
  import Sheet from '../../ui/Sheet.svelte';
  import Thumb from '../../ui/Thumb.svelte';
  import { createThumbState } from '../../ui/thumb.svelte.js';
  import { api } from '../../core/api.svelte.js';

  let { ids, onclose, ondone } = $props();

  let albums = $state([]);
  let name = $state('');
  let busy = $state(false);
  const thumbs = createThumbState();

  onMount(async () => {
      albums = await (await fetch(api('/api/albums'))).json();
  });

  async function addTo(id, label) {
    if (busy) return;
    busy = true;
    const res = await fetch(api(`/api/albums/${id}/assets`), { method: 'POST', body: JSON.stringify({ ids }) });
    const out = await res.json().catch(() => ({}));
    busy = false;
    if (res.ok) ondone(out.added ?? ids.length, label);
  }

  async function create(e) {
    e.preventDefault();
    const n = name.trim();
    if (!n || busy) return;
    const res = await fetch(api('/api/albums'), { method: 'POST', body: JSON.stringify({ name: n }) });
    if (res.ok) addTo((await res.json()).id, n);
  }
</script>

<Sheet {onclose} label="Tambahkan ke album" z={60} dim={0.45}>
    <h3>Tambahkan ke</h3>
    <form onsubmit={create}>
      <input bind:value={name} placeholder="Album baru" />
      <button class="go" disabled={!name.trim()}>Buat</button>
    </form>
    <ul>
      {#each albums as al (al.id)}
        <li>
          <button onclick={() => addTo(al.id, al.name)}>
            <span class="cov">
              {#if al.cover_id}
                <Thumb src={api(`/api/assets/${al.cover_id}/thumb`)} id={al.id} state={thumbs}>
                  {#snippet fallback()}<span class="cov-icon"><Icon name="album" size={24} /></span>{/snippet}
                </Thumb>
              {:else}
                <span class="cov-icon"><Icon name="album" size={24} /></span>
              {/if}
            </span>
            <span class="nm">{al.name}<small>{al.count} item</small></span>
          </button>
        </li>
      {/each}
    </ul>
</Sheet>

<style>
  h3 { font-size: 16px; padding: 0 12px 12px; }
  form { display: flex; gap: 8px; padding: 0 12px 12px; }
  input { flex: 1; height: 40px; padding: 0 12px; border-radius: 8px; border: 1px solid var(--line); background: var(--bg); color: var(--fg); font: inherit; }
  .go { padding: 0 16px; border-radius: 20px; color: var(--acc); font-weight: 500; }
  .go:disabled { opacity: .4; }
  ul { list-style: none; margin: 0; padding: 0; }
  li button { display: flex; align-items: center; justify-content: flex-start; gap: 12px; width: 100%; padding: 8px 12px; border-radius: 8px; text-align: left; }
  li button:hover { background: var(--hover); }
  .cov { width: 48px; height: 48px; border-radius: 8px; overflow: hidden; flex: none; }
  .cov-icon { display: grid; place-items: center; width: 100%; height: 100%; background: var(--tile); color: var(--mut); }
  .nm { display: flex; flex-direction: column; justify-content: center; align-items: flex-start; min-width: 0; text-align: left; }
  small { color: var(--mut); }
</style>
