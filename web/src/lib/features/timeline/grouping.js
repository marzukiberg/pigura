// Logika murni Timeline: pengelompokan per hari/bulan dan penataan baris (tanpa state/DOM).
import { justify, squares } from '../../core/layout.js';
import { dateFmt } from '../../core/format.js';

const today = new Date();

export const dayStart = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();

/** "Hari ini", "Kemarin", atau tanggal singkat (tahun hanya bila bukan tahun ini). */
export function dayLabel(t) {
  const d = new Date(t * 1000);
  const diff = Math.round((dayStart(today) - dayStart(d)) / 864e5);
  if (diff === 0) return 'Hari ini';
  if (diff === 1) return 'Kemarin';
  return (d.getFullYear() === today.getFullYear() ? dateFmt.weekdayShort : dateFmt.weekdayLong).format(d);
}

const SIZE_FACTOR = { sm: 0.72, md: 1, lg: 1.45 };

/** Tinggi baris target menurut lebar kontainer dan pilihan ukuran (kecil/sedang/besar). */
export const targetHeight = (width, size) =>
  Math.round((width < 600 ? 130 : width < 1100 ? 180 : 220) * SIZE_FACTOR[size]);

/**
 * Kelompokkan item berurutan. mode: 'day' | 'month' | 'none'.
 * Kelompok yang sama bisa muncul terpisah (mis. foto deep-link disisipkan di awal), jadi `key` dibuat unik.
 * @returns {{k: number, key: string, label: string, list: {a: any, i: number}[]}[]}
 */
export function groupItems(items, mode) {
  const out = [];
  const seen = new Map();
  items.forEach((a, i) => {
    const d = new Date(a.taken_at * 1000);
    const k = mode === 'none' ? 0 : mode === 'month' ? d.getFullYear() * 12 + d.getMonth() : dayStart(d);
    const last = out[out.length - 1];
    if (last && last.k === k) {
      last.list.push({ a, i });
      return;
    }
    const n = seen.get(k) ?? 0;
    seen.set(k, n + 1);
    out.push({
      k,
      key: n ? `${mode}:${k}#${n}` : `${mode}:${k}`,
      label: mode === 'none' ? '' : mode === 'month' ? dateFmt.month.format(d) : dayLabel(a.taken_at),
      list: [{ a, i }],
    });
  });
  return out;
}

/** Baris untuk satu kelompok sesuai layout ('justified' | 'square' | 'list'; daftar tidak memakai baris). */
export function layoutRows(list, layout, width, targetH, gap) {
  if (layout === 'list') return [];
  return layout === 'square' ? squares(list, width, targetH, gap) : justify(list, width, targetH, gap);
}
