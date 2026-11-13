<script>
  import Icon from '../../ui/Icon.svelte';

  let { count, onclear, onalbum, onfavorite, ondelete } = $props();
</script>

<div class="selbar">
  <div class="left">
    <button class="ib" onclick={onclear} aria-label="Batal memilih" title="Batal">
      <Icon name="close" size={20} />
    </button>
    <span class="count">{count} dipilih</span>
  </div>
  <div class="actions">
    <button class="pill primary" onclick={onalbum}>
      <Icon name="add" size={18} />
      <span>Album</span>
    </button>
    <button class="pill" onclick={onfavorite} title="Favorit">
      <Icon name="star" size={18} />
    </button>
    <button class="pill danger" onclick={ondelete} title="Hapus foto terpilih">
      <Icon name="trash" size={18} />
      <span>Hapus</span>
    </button>
  </div>
</div>

<style>
  .selbar {
    position: fixed;
    z-index: 40;
    inset: 0 0 auto 0;
    height: calc(64px + env(safe-area-inset-top));
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: env(safe-area-inset-top) 12px 0 16px;
    background: var(--surface);
    color: var(--fg);
    border-bottom: 1px solid var(--line);
    box-shadow: var(--elevated-shadow);
    animation: slide-down 0.18s ease-out;
  }
  @keyframes slide-down {
    from { transform: translateY(-100%); }
    to { transform: translateY(0); }
  }
  .left { display: flex; align-items: center; gap: 16px; }
  .count { font-size: 16px; font-weight: 500; white-space: nowrap; }
  .actions { display: flex; align-items: center; gap: 8px; }

  .ib {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: var(--mut);
    transition: background 0.15s, color 0.15s;
  }
  .ib:hover { background: var(--hover); color: var(--fg); }

  .pill {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 38px;
    padding: 0 16px;
    border-radius: 19px;
    background: var(--surface-variant);
    color: var(--fg);
    font-size: 13px;
    font-weight: 500;
    transition: all 0.15s;
  }
  .pill.primary { background: var(--acc); color: #ffffff; }
  .pill.danger { color: #ea4335; background: rgba(234, 67, 53, 0.1); }
  .pill.danger:hover { background: rgba(234, 67, 53, 0.2); }
  .pill:hover { filter: brightness(0.95); }

  /* Desktop: sisakan ruang sidebar */
  @media (min-width: 900px) { .selbar { left: 240px; } }

  @media (max-width: 480px) {
    .selbar { padding-right: 8px; }
    .left { gap: 4px; }
    .actions { gap: 6px; }
    .actions .pill { padding: 0 12px; }
    .actions .pill.danger span { display: none; } /* ikon sampah saja */
  }
</style>
