// Token akses opsional. Server hanya mewajibkannya bila PIGURA_TOKEN diisi.
import { api } from './api.svelte.js';

export const auth = $state({ required: false, ok: true, checked: false });

export async function checkAuth() {
  try {
    const res = await fetch(api('/api/auth/status'));
    if (res.ok) {
      const d = await res.json();
      auth.required = !!d.required;
      auth.ok = !!d.ok;
    }
  } catch {
    /* server belum terjangkau: biarkan alur normal menangani */
  } finally {
    auth.checked = true;
  }
}

// Mengembalikan pesan galat, atau null bila berhasil (cookie sesi HttpOnly diset server).
export async function login(token) {
  try {
    const res = await fetch(api('/api/auth/login'), { method: 'POST', body: JSON.stringify({ token }) });
    if (res.ok) {
      auth.ok = true;
      return null;
    }
    if (res.status === 429) return 'Terlalu banyak percobaan. Coba lagi sebentar.';
    return 'Kode akses salah.';
  } catch {
    return 'Server tidak terjangkau.';
  }
}

// Sesi habis di tengah pemakaian (401 dari /api/*) -> tampilkan layar login lagi.
let hooked = false;
export function installAuthHook() {
  if (hooked || typeof window === 'undefined') return;
  hooked = true;
  const orig = window.fetch.bind(window);
  window.fetch = async (...args) => {
    const res = await orig(...args);
    if (res.status === 401) {
      const u = typeof args[0] === 'string' ? args[0] : (args[0]?.url ?? '');
      if (u.includes('/api/') && !u.includes('/api/auth/')) {
        auth.required = true;
        auth.ok = false;
      }
    }
    return res;
  };
}
