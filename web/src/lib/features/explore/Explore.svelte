<script>
  import Icon from '../../ui/Icon.svelte';
  import Timeline from '../timeline/Timeline.svelte';
  import DateFilter from './DateFilter.svelte';
  import { api } from '../../core/api.svelte.js';

  let { searchQuery = $bindable(''), photoId = null, onphotochange } = $props();

  // Filter tanggal: { from, to } detik unix (to eksklusif) atau null.
  let dateRange = $state(null);

  let fetchUrl = $derived.by(() => {
    const r = dateRange;
    return (c) =>
      api(
        '/api/timeline?limit=100' +
          (r?.from != null ? `&from=${r.from}` : '') +
          (r?.to != null ? `&to=${r.to}` : '') +
          (c ? `&before=${c.before}&before_id=${c.before_id}` : '')
      );
  });

  const CHIPS = [
    { q: 'video', icon: 'play', label: 'Video' },
    { q: 'image', icon: 'photo', label: 'Foto' },
    { q: 'hari ini', icon: 'calendar', label: 'Hari ini' },
  ];
</script>

<div class="explore-page">
  <div class="explore-header">
    <h2>Jelajah</h2>
    <div class="chips">
      {#each CHIPS as c (c.q)}
        <button class="chip" class:selected={searchQuery === c.q} onclick={() => (searchQuery = searchQuery === c.q ? '' : c.q)}>
          <Icon name={c.icon} size={16} /> {c.label}
        </button>
      {/each}
    </div>
    <div class="chips">
      <DateFilter onchange={(r) => (dateRange = r)} />
    </div>
  </div>
  {#key JSON.stringify(dateRange)}
    <Timeline {searchQuery} {fetchUrl} {photoId} {onphotochange} empty="Tidak ada foto yang cocok dengan jelajah ini." />
  {/key}
</div>

<style>
  .explore-page {
    padding: 16px 12px 0;
  }

  .explore-header h2 {
    font-size: 22px;
    margin-bottom: 12px;
  }

  .chips {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    padding: 0 16px;
    border-radius: 18px;
    background: var(--surface-variant);
    color: var(--fg);
    font-size: 13px;
    font-weight: 500;
    border: 1px solid var(--line-subtle);
    transition: all 0.15s;
  }

  .chip:hover {
    background: var(--hover);
  }

  .chip.selected {
    background: var(--acc-bg);
    color: var(--acc-fg);
    border-color: var(--acc);
  }

  @media (max-width: 599px) {
    .explore-page { padding: 12px 8px 0; }
    .chips { gap: 6px; }
    .chip { height: 38px; padding: 0 14px; }
  }
</style>
