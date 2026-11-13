// Favorit: disimpan di localStorage (per browser), satu store bersama untuk semua daftar foto.
import { SvelteSet } from 'svelte/reactivity';
import { api } from './api.svelte.js';

const KEY = 'pigura-favorites';

export const favorites = new SvelteSet();

export function loadFavorites() {
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) || '[]');
    favorites.clear();
    if (Array.isArray(saved)) saved.forEach((id) => favorites.add(id));
  } catch {
    /* data rusak: abaikan */
  }
}

function save() {
  try {
    localStorage.setItem(KEY, JSON.stringify([...favorites]));
  } catch {
    /* penyimpanan penuh / mode privat */
  }
}

/** Balik status favorit satu foto; mengembalikan true bila kini favorit. */
export function toggleFavorite(id) {
  const now = !favorites.has(id);
  if (now) favorites.add(id);
  else favorites.delete(id);
  save();
  return now;
}

/** Atur status banyak foto sekaligus; mengembalikan true bila semuanya kini favorit. */
export function toggleFavorites(ids) {
  const allFav = ids.every((id) => favorites.has(id));
  ids.forEach((id) => (allFav ? favorites.delete(id) : favorites.add(id)));
  save();
  return !allFav;
}

/** id favorit langsung dari penyimpanan (valid sebelum daftar mana pun dimuat). */
export function storedFavoriteIds() {
  try {
    const ids = JSON.parse(localStorage.getItem(KEY) || '[]');
    return Array.isArray(ids) ? ids.filter((n) => Number.isInteger(n)) : [];
  } catch {
    return [];
  }
}

/**
 * URL timeline khusus favorit: server memfilter berdasarkan id sehingga semua favorit langsung
 * termuat (tanpa menelusuri seluruh timeline). Dipakai sebagai `fetchUrl` Timeline.
 */
export const favoritesFetchUrl = (cursor) =>
  api(
    '/api/timeline?limit=100&ids=' +
      (storedFavoriteIds().join(',') || '0') +
      (cursor ? `&before=${cursor.before}&before_id=${cursor.before_id}` : '')
  );
