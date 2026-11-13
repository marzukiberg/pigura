// Router History API minimal (tanpa dependensi).
//   /                 Foto
//   /photos/:id       Foto + lightbox
//   /explore          Jelajah        (lightbox: ?photo=ID)
//   /albums           Daftar album
//   /albums/:id       Isi album
//   /favorites        Favorit        (lightbox: ?photo=ID)
//   /duplicates       Cek duplikat
//   /s/<kunci>        Tautan berbagi publik (tanpa shell aplikasi)
//   /settings         Pengaturan
const state = $state({
  path: typeof window === 'undefined' ? '/' : window.location.pathname,
  search: typeof window === 'undefined' ? '' : window.location.search,
});

function sync() {
  state.path = window.location.pathname;
  state.search = window.location.search;
}

if (typeof window !== 'undefined') window.addEventListener('popstate', sync);

export function navigate(to, { replace = false } = {}) {
  const cur = window.location.pathname + window.location.search;
  if (to === cur) return;
  window.history[replace ? 'replaceState' : 'pushState']({}, '', to);
  sync();
}

// Handler klik untuk <a href>: navigasi tanpa reload, kecuali klik modifier/tab baru.
export function link(e) {
  if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
  const a = e.currentTarget;
  if (a.target && a.target !== '_self') return;
  const url = new URL(a.href, window.location.origin);
  if (url.origin !== window.location.origin) return;
  e.preventDefault();
  navigate(url.pathname + url.search);
}

export const VIEWS = ['photos', 'explore', 'albums', 'favorites', 'duplicates', 'settings'];

// path + search -> { view, photoId, albumId }
export function parse(path, search = '') {
  const clean = path.replace(/\/+$/, '') || '/';
  const q = new URLSearchParams(search);
  const qPhoto = Number(q.get('photo')) || null;
  let m;
  if (clean === '/') return { view: 'photos', photoId: null, albumId: null };
  if ((m = clean.match(/^\/photos\/(\d+)$/))) return { view: 'photos', photoId: Number(m[1]), albumId: null };
  if (clean === '/explore') return { view: 'explore', photoId: qPhoto, albumId: null };
  if (clean === '/favorites') return { view: 'favorites', photoId: qPhoto, albumId: null };
  if ((m = clean.match(/^\/s\/([A-Za-z0-9_-]{22})$/))) return { view: 'share', photoId: null, albumId: null, shareKey: m[1] };
  if (clean === '/duplicates') return { view: 'duplicates', photoId: null, albumId: null };
  if (clean === '/settings') return { view: 'settings', photoId: null, albumId: null };
  if (clean === '/albums') return { view: 'albums', photoId: null, albumId: null };
  if ((m = clean.match(/^\/albums\/(\d+)$/))) return { view: 'albums', photoId: null, albumId: Number(m[1]) };
  return { view: 'notfound', photoId: null, albumId: null };
}

export const route = {
  get path() { return state.path; },
  get search() { return state.search; },
  get current() { return parse(state.path, state.search); },
};
