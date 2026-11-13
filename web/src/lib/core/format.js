// Pemformat bersama (locale id-ID): ukuran file, durasi, tanggal.

const KB = 1 << 10;
const MB = 1 << 20;
const GB = 1 << 30;

/** Ukuran ringkas: "512 B", "84 KB", "2,1 MB", "1,25 GB". */
export function fmtSize(n) {
  if (n >= GB) return (n / GB).toFixed(2) + ' GB';
  if (n >= MB) return (n / MB).toFixed(1) + ' MB';
  if (n >= KB) return Math.round(n / KB) + ' KB';
  return n + ' B';
}

/** Ukuran untuk panel info: satu desimal di semua satuan, "Tidak diketahui" bila kosong. */
export function fmtBytesDetail(bytes) {
  if (!bytes || bytes <= 0) return 'Tidak diketahui';
  const units = ['B', 'KB', 'MB', 'GB'];
  let i = 0;
  let val = bytes;
  while (val >= KB && i < units.length - 1) {
    val /= KB;
    i++;
  }
  return `${val.toFixed(1)} ${units[i]}`;
}

/** Durasi video "m:ss"; string kosong bila tak ada. */
export function formatDuration(sec) {
  if (!sec || sec <= 0) return '';
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
}

/** Bitrate ringkas: "14,5 Mbps" / "820 kbps". */
export function fmtBitrate(bps) {
  if (!bps || bps <= 0) return '';
  return bps >= 1e6 ? `${(bps / 1e6).toFixed(1)} Mbps` : `${Math.round(bps / 1e3)} kbps`;
}

const mk = (opts) => new Intl.DateTimeFormat('id-ID', opts);

/** Pemformat tanggal siap pakai (dibuat sekali; Intl.DateTimeFormat mahal dibangun). */
export const dateFmt = {
  /** 12 Nov 2026, 14.30 */
  medium: mk({ dateStyle: 'medium', timeStyle: 'short' }),
  /** 12 November 2026 pukul 14.30 */
  long: mk({ dateStyle: 'long', timeStyle: 'short' }),
  /** Kamis, 12 November 2026 pukul 14.30 */
  full: mk({ dateStyle: 'full', timeStyle: 'short' }),
  /** 12 Nov 2026 */
  day: mk({ day: 'numeric', month: 'short', year: 'numeric' }),
  /** 12 Nov 2026, 14.30 (hari-bulan-tahun + jam) */
  dayTime: mk({ day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }),
  /** Kam, 12 Nov */
  weekdayShort: mk({ weekday: 'short', day: 'numeric', month: 'short' }),
  /** Kam, 12 Nov 2026 */
  weekdayLong: mk({ weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' }),
  /** November 2026 */
  month: mk({ month: 'long', year: 'numeric' }),
};

/** Nama file dari path relatif. */
export const baseName = (path) => (path ? path.split('/').pop() : '');
