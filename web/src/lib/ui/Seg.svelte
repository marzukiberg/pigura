<script>
  import Icon from './Icon.svelte';

  // Kontrol segmen (pilih satu). options: [{ id, label, icon?, count? }]
  //   role     : 'radio' (pengaturan) | 'tab' (menyaring konten)
  //   variant  : 'fill' (lebar terbagi rata) | 'inline' (selebar isi) | 'wrap' (boleh baris ganda)
  //   size     : 'xs' 32px | 'md' 36px | 'sm' 34px (bulat lebih besar, mis. daftar album/tema)
  let { options, value, onchange, label = '', role = 'radio', variant = 'fill', size = 'md', px = 16, class: cls = '' } = $props();
</script>

<div class="seg {variant} {size} {cls}" style:--px="{px}px" role={role === 'tab' ? 'tablist' : 'radiogroup'} aria-label={label}>
  {#each options as o (o.id)}
    <button
      type="button"
      {role}
      aria-checked={role === 'radio' ? value === o.id : undefined}
      aria-selected={role === 'tab' ? value === o.id : undefined}
      class:on={value === o.id}
      onclick={() => onchange(o.id)}
    >
      {#if o.icon}<Icon name={o.icon} size={16} />{/if}
      <span>{o.label}</span>
      {#if o.count !== undefined}<span class="n">{o.count}</span>{/if}
    </button>
  {/each}
</div>

<style>
  .seg {
    display: flex;
    padding: 3px;
    border-radius: 12px;
    background: var(--surface-variant);
    --h: 36px;
    --rb: 9px;
  }
  .seg.inline { display: inline-flex; }
  .seg.wrap { flex-wrap: wrap; gap: 4px; }
  .seg.xs { --h: 32px; }
  .seg.sm { --h: 34px; --rb: 11px; border-radius: 14px; }

  button {
    flex: 1 1 0;
    gap: 6px;
    height: var(--h);
    padding: 0 10px;
    border-radius: var(--rb);
    font-size: 13px;
    font-weight: 500;
    color: var(--mut);
    transition: background 0.15s, color 0.15s;
  }
  .inline button { flex: none; padding: 0 var(--px); }
  .wrap button { flex: 1 1 auto; }
  button:hover:not(.on) { color: var(--fg); }
  button.on { background: var(--surface); color: var(--fg); box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2); }
  .n { font-size: 11px; opacity: 0.7; }
  @media (pointer: coarse) { .seg.xs { --h: 36px; } }
</style>
