<script>
  import { copyText } from '../../core/clipboard.js';
  import { onMount } from 'svelte';
  import Icon from '../../ui/Icon.svelte';
  import { dateFmt } from '../../core/format.js';
  import { showToast } from '../../core/toast.svelte.js';
  import { api } from '../../core/api.svelte.js';

  let shares = $state([]);
  let publicOk = $state(false);
  let ready = $state(false);
  const fmt = dateFmt.medium;
  const STATUS = { active: 'Aktif', expired: 'Kedaluwarsa', revoked: 'Dicabut' };

  async function load() {
    try {
      const res = await fetch(api('/api/shares'));
      if (res.ok) {
        const d = await res.json();
        shares = d.items;
        publicOk = d.public;
      }
    } finally {
      ready = true;
    }
  }

  const link = (s) => (publicOk ? s.url : window.location.origin + s.path);

  const flash = (m) => showToast(m);

  async function copy(s) {
    flash((await copyText(link(s))) ? 'Tautan disalin' : 'Gagal menyalin');
  }

  async function revoke(s) {
    if (!confirm(`Cabut tautan “${s.title || 'foto'}”?\n\nTautan langsung berhenti bisa dibuka.`)) return;
    const res = await fetch(api(`/api/shares/${s.id}`), { method: 'DELETE' });
    flash(res.ok ? 'Tautan dicabut' : 'Gagal mencabut');
    load();
  }

  onMount(load);
</script>

<section>
  <h2>Tautan dibagikan</h2>
  {#if ready && shares.length === 0}
    <p class="empty">Belum ada. Buka foto atau album, lalu pilih ikon bagikan.</p>
  {/if}
  <ul>
    {#each shares as s (s.id)}
      <li class:off={s.status !== 'active'}>
        <span class="ico"><Icon name={s.kind === 'album' ? 'album' : 'photo'} size={20} /></span>
        <div class="meta">
          <div class="nm">{s.title || 'Foto'}</div>
          <div class="sub">
            <span class="st st-{s.status}">{STATUS[s.status]}</span>
            {s.expires_at ? `sampai ${fmt.format(new Date(s.expires_at * 1000))}` : 'tanpa batas'} · {s.views}× dilihat
            {#if s.has_password} · password{/if}{#if s.allow_download} · unduh{/if}
          </div>
        </div>
        {#if s.status === 'active'}
          <button class="act" onclick={() => copy(s)} aria-label="Salin tautan" title="Salin tautan"><Icon name="copy" size={18} /></button>
          <button class="act del" onclick={() => revoke(s)} aria-label="Cabut tautan" title="Cabut"><Icon name="trash" size={18} /></button>
        {/if}
      </li>
    {/each}
  </ul>
  {#if ready && shares.length > 0 && !publicOk}
    <p class="note">Alamat publik belum dikonfigurasi: tautan hanya bisa dibuka dari jaringan privat Anda.</p>
  {/if}
</section>

<style>
  section { width: 100%; display: flex; flex-direction: column; gap: 10px; }
  h2 { font-size: 13px; font-weight: 600; color: var(--mut); text-transform: uppercase; letter-spacing: 0.06em; }
  ul { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
  li { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: 14px; background: var(--surface-variant); }
  li.off { opacity: 0.6; }
  .ico { width: 36px; height: 36px; border-radius: 10px; display: grid; place-items: center; background: var(--surface); color: var(--mut); flex: none; }
  .meta { flex: 1; min-width: 0; }
  .nm { font-size: 14px; font-weight: 500; color: var(--fg); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .sub { font-size: 12px; color: var(--mut); line-height: 1.4; }
  .st { font-weight: 600; margin-right: 4px; }
  .st-active { color: #188038; } .st-expired { color: #b06000; } .st-revoked { color: #d93025; }
  .act { width: 40px; height: 40px; border-radius: 50%; color: var(--mut); flex: none; }
  .act:hover { background: var(--hover); color: var(--fg); }
  .act.del:hover { color: #d93025; }
  .empty, .note { margin: 0; font-size: 13px; color: var(--mut); }
</style>
