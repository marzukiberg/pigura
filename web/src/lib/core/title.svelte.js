// Judul tab browser mengikuti halaman + path. App menyusun judul akhir dari:
//   [foto di lightbox] · [nama album] · [nama halaman] · Pigura
export const APP_NAME = 'Pigura';

export const titleState = $state({ photo: '', album: '' });

export const PAGE_TITLES = {
  photos: '', // beranda: cukup "Pigura"
  explore: 'Jelajah',
  albums: 'Album',
  favorites: 'Favorit',
  duplicates: 'Duplikat',
  settings: 'Pengaturan',
  share: 'Foto dibagikan',
  notfound: 'Halaman tidak ditemukan',
};

export function buildTitle(view, { photo = '', album = '' } = {}) {
  return [photo, album, PAGE_TITLES[view] ?? '', APP_NAME].filter(Boolean).join(' · ');
}
