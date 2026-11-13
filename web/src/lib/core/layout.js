// Layout baris rata (justified) ala Google Photos.
// Tiap baris diisi item sampai penuh lalu diskalakan agar tepat selebar container.

const MIN_AR = 0.5; // batasi rasio ekstrem (panorama / potret sangat tinggi)
const MAX_AR = 2.5;

export function aspect(a) {
  const r = a.width > 0 && a.height > 0 ? a.width / a.height : 1;
  return Math.min(MAX_AR, Math.max(MIN_AR, r));
}

/**
 * @param {{a: any, i: number}[]} list item (a = aset, i = indeks global)
 * @returns {{h: number, tiles: {a: any, i: number, w: number}[]}[]}
 */
export function justify(list, width, targetH, gap) {
  if (width <= 0) return [];
  const rows = [];
  let row = [];
  let sum = 0;
  const flush = (h) => rows.push({ h, tiles: row.map((t) => ({ a: t.a, i: t.i, w: t.ar * h })) });

  for (const it of list) {
    row.push({ ...it, ar: aspect(it.a) });
    sum += row[row.length - 1].ar;
    const gaps = gap * (row.length - 1);
    if (sum * targetH + gaps >= width) {
      flush((width - gaps) / sum);
      row = [];
      sum = 0;
    }
  }
  if (row.length) flush(targetH); // baris terakhir tidak diregangkan
  return rows;
}

/**
 * Grid kotak: jumlah kolom mengikuti ukuran target, sel selalu persegi.
 * @returns {{h: number, tiles: {a: any, i: number, w: number}[]}[]}
 */
export function squares(list, width, target, gap) {
  if (width <= 0) return [];
  const cols = Math.max(2, Math.round((width + gap) / (target + gap)));
  const cell = (width - gap * (cols - 1)) / cols;
  const rows = [];
  for (let i = 0; i < list.length; i += cols) {
    rows.push({ h: cell, tiles: list.slice(i, i + cols).map((t) => ({ a: t.a, i: t.i, w: cell })) });
  }
  return rows;
}
