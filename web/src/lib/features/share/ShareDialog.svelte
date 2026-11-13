<script>
  import { copyText } from '../../core/clipboard.js';
  import Icon from '../../ui/Icon.svelte';
  import Sheet from '../../ui/Sheet.svelte';
  import Seg from '../../ui/Seg.svelte';
  import { dateFmt } from '../../core/format.js';
  import { api } from '../../core/api.svelte.js';

  // kind: 'asset' | 'album'
  let { kind, targetId, title = '', onclose } = $props();

  const EXPIRY = [
    { id: 3600, label: '1 jam' },
    { id: 86400, label: '1 hari' },
    { id: 7 * 86400, label: '7 hari' },
    { id: 30 * 86400, label: '30 hari' },
    { id: 0, label: 'Tanpa batas' },
  ];

  let expiresIn = $state(7 * 86400);
  let password = $state('');
  let allowDownload = $state(false);
  let busy = $state(false);
  let error = $state('');
  let result = $state(null); // { url, expires_at, has_password, allow_download, public }
  let copied = $state(false);
  let input;

  const noun = $derived(kind === 'album' ? `album “${title}”` : 'foto ini');
  const fmt = dateFmt.medium;

  async function create() {
    if (busy) return;
    if (password && password.length < 4) {
      error = 'Password minimal 4 karakter.';
      return;
    }
    busy = true;
    error = '';
    try {
      const res = await fetch(api('/api/shares'), {
        method: 'POST',
        body: JSON.stringify({
          kind,
          target_id: targetId,
          expires_in: expiresIn,
          password: password || undefined,
          allow_download: allowDownload,
        }),
      });
      const out = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(out.error || 'Gagal membuat tautan.');
      result = { ...out, url: out.public ? out.url : window.location.origin + out.path };
    } catch (e) {
      error = e.message || 'Gagal membuat tautan.';
    } finally {
      busy = false;
    }
  }

  async function copy() {
    if (!(await copyText(result.url))) {
      input?.select();
      document.execCommand?.('copy');
    }
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }

  const canShare = typeof navigator !== 'undefined' && !!navigator.share;
  async function nativeShare() {
    try {
      await navigator.share({ title: 'Foto dibagikan', url: result.url });
    } catch {
      /* dibatalkan */
    }
  }
</script>

<Sheet {onclose} label="Bagikan" title={`Bagikan ${noun}`} width="460px" maxHeight="90dvh" radius="20px" padding="16px" padBottom="18px" gap="14px" z={120}>
    {#if !result}
      <div class="sec">
        <div class="lbl">Berlaku</div>
        <Seg options={EXPIRY} value={expiresIn} onchange={(v) => (expiresIn = v)} variant="wrap" label="Berlaku" />
      </div>

      <div class="sec">
        <label class="lbl" for="sh-pw">Password (opsional)</label>
        <input id="sh-pw" type="password" bind:value={password} autocomplete="new-password" placeholder="Kosongkan bila tanpa password" />
      </div>

      <label class="check">
        <input type="checkbox" bind:checked={allowDownload} />
        <span>
          Izinkan unduh file asli
          <small>File asli membawa data lokasi (GPS) dan perangkat. Tanpa ini pengunjung hanya melihat versi yang sudah dibersihkan.</small>
        </span>
      </label>

      {#if error}<p class="err" role="alert">{error}</p>{/if}
      <button class="go" onclick={create} disabled={busy}>{busy ? 'Membuat…' : 'Buat tautan'}</button>
    {:else}
      <div class="sec">
        <div class="lbl">Tautan</div>
        <div class="row">
          <input bind:this={input} readonly value={result.url} onfocus={(e) => e.currentTarget.select()} aria-label="Tautan berbagi" />
          <button class="copy" onclick={copy}>
            <Icon name={copied ? 'check' : 'copy'} size={18} /><span>{copied ? 'Disalin' : 'Salin'}</span>
          </button>
        </div>
        {#if canShare}<button class="share" onclick={nativeShare}><Icon name="share" size={18} /><span>Bagikan lewat…</span></button>{/if}
      </div>

      <ul class="facts">
        <li>{result.expires_at ? `Berlaku sampai ${fmt.format(new Date(result.expires_at * 1000))}` : 'Tidak kedaluwarsa'}</li>
        <li>{result.has_password ? 'Dilindungi password' : 'Tanpa password: siapa pun yang punya tautan bisa melihat'}</li>
        <li>{result.allow_download ? 'Unduhan file asli diizinkan' : 'Hanya lihat (tanpa unduh file asli)'}</li>
      </ul>
      {#if !result.public}
        <p class="warn" role="note">
          Alamat publik belum dikonfigurasi, jadi tautan ini hanya bisa dibuka dari jaringan privat Anda.
        </p>
      {/if}
      <p class="hint">Cabut kapan saja di Pengaturan, Tautan dibagikan.</p>
      <button class="go" onclick={onclose}>Selesai</button>
    {/if}
</Sheet>

<style>
  .sec { display: flex; flex-direction: column; gap: 6px; }
  .lbl { font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--mut); }
  input[type='password'], .row input {
    height: 44px; padding: 0 12px; border-radius: 12px; border: 1px solid var(--line); background: var(--bg); color: var(--fg); font: inherit; width: 100%; min-width: 0;
  }
  input:focus { outline: 2px solid var(--acc); border-color: transparent; }
  .check { display: flex; gap: 10px; align-items: flex-start; font-size: 14px; cursor: pointer; }
  .check input { width: 20px; height: 20px; margin-top: 1px; accent-color: var(--acc); flex: none; }
  .check small { display: block; color: var(--mut); font-size: 12px; line-height: 1.4; margin-top: 2px; }
  .go { height: 46px; border-radius: 23px; background: var(--acc-bg); color: var(--acc-fg); font-weight: 500; font-size: 15px; }
  .go:disabled { opacity: 0.5; }
  .row { display: flex; gap: 8px; }
  .copy, .share { gap: 6px; height: 44px; padding: 0 16px; border-radius: 22px; background: var(--acc-bg); color: var(--acc-fg); font-weight: 500; font-size: 14px; flex: none; }
  .share { align-self: flex-start; background: var(--surface-variant); color: var(--fg); }
  .facts { margin: 0; padding-left: 18px; font-size: 13px; color: var(--fg-variant); line-height: 1.6; }
  .err { margin: 0; color: #d93025; font-size: 13px; }
  .warn { margin: 0; padding: 10px 12px; border-radius: 12px; background: rgba(251, 188, 4, 0.15); color: var(--fg); font-size: 13px; line-height: 1.45; }
  .hint { margin: 0; font-size: 12px; color: var(--mut); }
</style>
