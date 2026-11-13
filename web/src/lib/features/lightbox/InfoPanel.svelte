<script>
  import { copyText } from '../../core/clipboard.js';
  import Icon from '../../ui/Icon.svelte';
  import { api } from '../../core/api.svelte.js';
  import { fmtBytesDetail, fmtBitrate, formatDuration, dateFmt, baseName } from '../../core/format.js';
  import { showToast } from '../../core/toast.svelte.js';

  // Panel detail foto (sisi kanan di desktop, layar penuh di ponsel).
  // a: aset; when: waktu file terformat; onclose: tutup panel.
  let { a, when, onclose } = $props();

  let info = $state(null); // metadata dari server (EXIF/GPS/codec); null = belum ada
  let loading = $state(false);

  // Ambil metadata tiap ganti foto; hasil lama dibatalkan agar tidak menimpa foto yang baru.
  $effect(() => {
    const id = a.id;
    const ctrl = new AbortController();
    info = null;
    loading = true;
    fetch(api(`/api/assets/${id}/metadata`), { signal: ctrl.signal })
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => {
        info = d;
        loading = false;
      })
      .catch((e) => {
        if (e.name !== 'AbortError') loading = false;
      });
    return () => ctrl.abort();
  });

  const ext = (path) => (path?.split('.').pop() || 'JPEG').toUpperCase();
  const dims = $derived({ w: info?.width || a.width, h: info?.height || a.height });
  const resolution = $derived(
    dims.w > 0 && dims.h > 0
      ? (a.type === 'video' ? '' : `${((dims.w * dims.h) / 1e6).toFixed(1)} MP • `) + `${dims.w} × ${dims.h}`
      : ''
  );

  // "2026-02-22 18:33:49" (waktu dinding tanpa zona) -> tampil apa adanya, tanpa konversi zona.
  const taken = $derived.by(() => {
    const m = info?.taken_local?.match(/^(\d+)-(\d+)-(\d+) (\d+):(\d+):(\d+)$/);
    if (!m) return '';
    const d = new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +m[6]);
    return dateFmt.full.format(d) + (info.taken_offset ? ` (${info.taken_offset})` : '');
  });

  const videoTaken = $derived(info?.video?.created ? dateFmt.full.format(new Date(info.video.created)) : '');
  const takenText = $derived(taken || videoTaken);
  const takenSource = $derived(taken ? 'EXIF' : 'metadata video');
  const takenSub = $derived(
    !takenText
      ? 'Waktu pengambilan file'
      : takenText === when
        ? `Waktu pengambilan (${takenSource})`
        : `Waktu pengambilan (${takenSource}) • file: ${when}`
  );

  const cam = $derived(info?.camera);
  const camName = $derived(cam ? [cam.make, cam.model].filter(Boolean).join(' ') : '');
  const ex = $derived(info?.exposure);
  const exMain = $derived(
    ex
      ? [
          ex.f_number ? `ƒ/${ex.f_number}` : '',
          ex.shutter ? `${ex.shutter} dtk` : '',
          ex.iso ? `ISO ${ex.iso}` : '',
        ]
          .filter(Boolean)
          .join(' • ')
      : ''
  );
  const exSub = $derived(
    ex
      ? [
          ex.focal_length ? `${ex.focal_length} mm${ex.focal_35 ? ` (setara ${ex.focal_35} mm)` : ''}` : '',
          ex.flash ? `Flash: ${ex.flash}` : '',
          ex.bias && ex.bias !== '0.00 EV' ? `EV ${ex.bias}` : '',
          ex.metering,
          ex.program,
          ex.white_balance ? `WB ${ex.white_balance}` : '',
        ]
          .filter(Boolean)
          .join(' • ')
      : ''
  );

  const gps = $derived(info?.gps);
  const coords = $derived(gps ? `${gps.lat.toFixed(6)}, ${gps.lon.toFixed(6)}` : '');
  const mapUrl = $derived(
    gps ? `https://www.openstreetmap.org/?mlat=${gps.lat}&mlon=${gps.lon}#map=17/${gps.lat}/${gps.lon}` : ''
  );

  const vid = $derived(info?.video);
  const vidMain = $derived(
    vid ? [vid.codec?.toUpperCase(), info.width && `${info.width} × ${info.height}`, vid.fps && `${vid.fps} fps`].filter(Boolean).join(' • ') : ''
  );
  const vidSub = $derived(
    vid
      ? [
          vid.duration ? `Durasi ${formatDuration(vid.duration)}` : '',
          fmtBitrate(vid.bitrate),
          vid.audio_codec ? `audio ${vid.audio_codec.toUpperCase()}` : 'tanpa audio',
        ]
          .filter(Boolean)
          .join(' • ')
      : ''
  );

  async function copyCoords() {
    showToast((await copyText(coords)) ? 'Koordinat disalin' : 'Gagal menyalin');
  }
</script>

<div class="drawer" role="complementary" aria-label="Detail info foto">
  <div class="drawer-header">
    <h3>Detail</h3>
    <button class="close" onclick={onclose} aria-label="Tutup detail">
      <Icon name="close" size={20} />
    </button>
  </div>

  <div class="drawer-body">
    <div class="info-section">
      <div class="info-row">
        <span class="info-icon"><Icon name="calendar" size={20} /></span>
        <div class="info-content">
          <div class="info-primary">{takenText || when}</div>
          <div class="info-sub">{takenSub}</div>
        </div>
      </div>

      <div class="info-row">
        <span class="info-icon"><Icon name={a.type === 'video' ? 'film' : 'photo'} size={20} /></span>
        <div class="info-content">
          <div class="info-primary">{baseName(a.path) || 'Foto'}</div>
          <div class="info-sub">
            {resolution}
            {#if a.size > 0}• {fmtBytesDetail(a.size)}{/if}
            • {info?.format ? info.format.toUpperCase() : ext(a.path)}
          </div>
        </div>
      </div>

      {#if camName || cam?.lens || cam?.software}
        <div class="info-row">
          <span class="info-icon"><Icon name="camera" size={20} /></span>
          <div class="info-content">
            <div class="info-primary">{camName || cam.lens}</div>
            {#if camName && (cam.lens || cam.software)}
              <div class="info-sub">{[cam.lens, cam.software].filter(Boolean).join(' • ')}</div>
            {/if}
          </div>
        </div>
      {/if}

      {#if exMain || exSub}
        <div class="info-row">
          <span class="info-icon"><Icon name="aperture" size={20} /></span>
          <div class="info-content">
            <div class="info-primary">{exMain || exSub}</div>
            {#if exMain && exSub}<div class="info-sub">{exSub}</div>{/if}
          </div>
        </div>
      {/if}

      {#if vid}
        <div class="info-row">
          <span class="info-icon"><Icon name="film" size={20} /></span>
          <div class="info-content">
            <div class="info-primary">{vidMain}</div>
            <div class="info-sub">{vidSub}</div>
          </div>
        </div>
      {/if}

      {#if gps}
        <div class="info-row">
          <span class="info-icon"><Icon name="pin" size={20} /></span>
          <div class="info-content">
            <div class="info-primary">{coords}</div>
            <div class="info-sub">
              {#if gps.alt != null}Ketinggian {gps.alt} m • {/if}
              <a href={mapUrl} target="_blank" rel="noopener noreferrer">Buka di peta</a>
              • <button class="linkbtn" onclick={copyCoords}>Salin</button>
            </div>
          </div>
        </div>
      {/if}

      {#if loading}
        <div class="info-row" aria-busy="true">
          <span class="info-icon"><Icon name="info" size={20} /></span>
          <div class="info-content"><div class="info-sub">Memuat detail…</div></div>
        </div>
      {/if}

      <div class="info-row">
        <span class="info-icon"><Icon name="storage" size={20} /></span>
        <div class="info-content">
          <div class="info-primary">Tersimpan di server</div>
          <div class="info-sub">{a.path || 'Root library'}</div>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .drawer {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: 360px;
    z-index: 20;
    background: #1e1f20;
    color: #e3e3e3;
    border-left: 1px solid #3c4043;
    box-shadow: -4px 0 24px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    animation: slide-in-right 0.22s cubic-bezier(0.4, 0, 0.2, 1);
  }
  @keyframes slide-in-right {
    from { transform: translateX(100%); }
    to { transform: translateX(0); }
  }
  @media (max-width: 768px) {
    .drawer { width: 100%; }
  }

  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 64px;
    padding: 0 16px;
    padding-top: env(safe-area-inset-top);
    border-bottom: 1px solid #3c4043;
  }
  .drawer-header h3 { font-size: 18px; font-weight: 500; }
  .close {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    color: #ffffff;
    transition: background 0.15s;
  }
  .close:hover { background: rgba(255, 255, 255, 0.15); }

  .drawer-body { flex: 1; overflow-y: auto; padding: 16px; }
  .info-section { display: flex; flex-direction: column; gap: 20px; }
  .info-row { display: flex; gap: 16px; align-items: flex-start; }
  .info-icon { color: #8e918f; display: flex; align-items: center; padding-top: 2px; flex: none; }
  .info-content { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
  .info-primary { font-size: 14px; font-weight: 500; color: #e3e3e3; word-break: break-word; }
  .info-sub { font-size: 12px; color: #8e918f; word-break: break-word; line-height: 1.5; }
  .info-sub a, .linkbtn { color: #8ab4f8; text-decoration: none; font-size: inherit; display: inline; padding: 0; }
  .info-sub a:hover, .linkbtn:hover { text-decoration: underline; }
</style>
