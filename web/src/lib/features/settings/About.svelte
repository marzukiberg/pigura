<script>
  import { APP_VERSION, REPO_URL, changelog } from '../../core/changelog.js';

  const GROUP_LABELS = { Added: 'Baru', Changed: 'Diubah', Fixed: 'Diperbaiki', Removed: 'Dihapus', Security: 'Keamanan', Deprecated: 'Usang' };
  const label = (t) => GROUP_LABELS[t] ?? t;
  // Rilis (0.2.0) diberi awalan v; build dev/antar-tag (dev, 3ce6995, 0.2.0-3-gabc-dirty) tampil apa adanya.
  const shown = /^\d+\.\d+\.\d+$/.test(APP_VERSION) ? `v${APP_VERSION}` : APP_VERSION;
</script>

<section class="about">
  <h2>Tentang</h2>

  <div class="app">
    <div class="name">Pigura <span class="ver">{shown}</span></div>
    <p>Penampil foto &amp; video self-hosted yang ringan. “Pigura” berarti bingkai foto.</p>
    <p class="links">
      <a href={REPO_URL} target="_blank" rel="noopener noreferrer">GitHub</a>
      <a href="{REPO_URL}/issues" target="_blank" rel="noopener noreferrer">Laporkan masalah</a>
      <a href="{REPO_URL}/blob/main/LICENSE" target="_blank" rel="noopener noreferrer">Lisensi MIT</a>
    </p>
  </div>

  <h3>Riwayat perubahan</h3>
  <div class="log">
    {#each changelog as r, i (r.version)}
      <details open={i === 0}>
        <summary>
          <span class="rv">{r.version === 'Unreleased' ? 'Belum dirilis' : `v${r.version}`}</span>
          {#if r.date}<span class="rd">{r.date}</span>{/if}
        </summary>
        {#each r.groups.filter((g) => g.items.length) as g (g.title)}
          <div class="gt">{label(g.title)}</div>
          <ul>
            {#each g.items as it}<li>{it}</li>{/each}
          </ul>
        {/each}
      </details>
    {/each}
  </div>
</section>

<style>
  .app { display: flex; flex-direction: column; gap: 6px; }
  .name { font-size: 18px; font-weight: 500; color: var(--fg); }
  .ver { margin-left: 6px; padding: 2px 8px; border-radius: 10px; background: var(--surface-variant); color: var(--mut); font-size: 12px; font-weight: 500; }
  p { margin: 0; font-size: 13px; color: var(--fg-variant); line-height: 1.5; }
  .links { display: flex; flex-wrap: wrap; gap: 6px 16px; }
  a { color: var(--acc); text-decoration: none; }
  a:hover { text-decoration: underline; }
  /* sama dengan judul section lain di Settings (gaya itu scoped, tidak menjangkau komponen ini) */
  h2 { font-size: 13px; font-weight: 600; color: var(--mut); text-transform: uppercase; letter-spacing: 0.06em; margin: 0; }
  h3 { margin: 8px 0 0; font-size: 13px; font-weight: 600; color: var(--mut); }
  .about { display: flex; flex-direction: column; gap: 12px; }
  .log { display: flex; flex-direction: column; gap: 8px; align-self: stretch; }
  details { border: 1px solid var(--line); border-radius: 12px; padding: 10px 14px; }
  summary { cursor: pointer; display: flex; align-items: baseline; gap: 10px; color: var(--fg); font-size: 14px; }
  .rv { font-weight: 500; }
  .rd { color: var(--mut); font-size: 12px; }
  .gt { margin: 12px 0 4px; font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--mut); }
  ul { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 3px; }
  li { font-size: 13px; color: var(--fg-variant); line-height: 1.45; }
</style>
