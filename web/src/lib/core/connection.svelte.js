// Koneksi ke server: alamat + folder scan tersimpan, pengaturan awal, dan penyelarasan origin.
import { api, connection } from './api.svelte.js';
import { auth, checkAuth } from './auth.svelte.js';
import { checkHealth } from './health.svelte.js';

const KEY = 'pigura-connection';

export const setup = $state({
  publicOnly: false, // origin ini hanya melayani tautan berbagi (listener publik): tidak ada form koneksi
  ready: false, // deteksi koneksi awal selesai
  configured: false,
  auto: false, // UI dilayani server yang sama dengan API (produksi): tanpa isi URL
  needsLibrary: false, // server belum punya folder library -> minta folder saja
  scanBase: '',
  error: '',
  saving: false,
  saved: false,
});

const sameOrigin = () => connection.baseUrl.replace(/\/+$/, '') === window.location.origin;

/** Apa yang dilayani origin ini? 'owner' = API pigura, 'public' = hanya tautan berbagi, 'none'. */
async function probeOrigin() {
  try {
    const res = await fetch(`${window.location.origin}/api/auth/status`);
    const d = res.ok ? await res.json() : null;
    if (d?.public_only === true) return 'public';
    return d && typeof d.required === 'boolean' ? 'owner' : 'none';
  } catch {
    return 'none';
  }
}

/**
 * Koneksi awal. Alamat tersimpan dipakai (diselaraskan ke origin bila perlu); bila belum ada dan origin
 * melayani API (produksi: UI tertanam di server, atau dev lewat proxy Vite) langsung terhubung ke sana
 * tanpa pengaturan awal. Form URL hanya muncul bila UI dipisah dari server (mis. aplikasi Capacitor).
 */
export async function initConnection() {
  loadSavedConnection();
  if (setup.configured) {
    await reconcileBase();
  } else {
    const kind = await probeOrigin();
    if (kind === 'owner') {
      connection.baseUrl = window.location.origin;
      setup.configured = true;
    } else if (kind === 'public') {
      setup.publicOnly = true;
    }
  }
  setup.auto = setup.configured && sameOrigin();
  setup.ready = true;
}

/** Ambil folder library & base scan dari server (setelah login bila token aktif). */
export async function loadServerConfig() {
  try {
    const res = await fetch(api('/api/config'));
    if (!res.ok) return;
    const d = await res.json();
    setup.scanBase = d.scan_base || '';
    if (d.library) connection.library = d.library;
    setup.needsLibrary = !d.library;
  } catch {
    /* abaikan: server belum terjangkau */
  }
}

export function loadSavedConnection() {
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) || 'null');
    if (saved?.baseUrl && saved?.library) {
      connection.baseUrl = saved.baseUrl;
      connection.library = saved.library;
      setup.configured = true;
    }
  } catch {
    localStorage.removeItem(KEY);
  }
}

const persist = (baseUrl, library) => localStorage.setItem(KEY, JSON.stringify({ baseUrl, library }));

/**
 * Alamat tersimpan bisa tak sama dengan tempat UI disajikan (mis. 127.0.0.1 vs localhost, atau URL lain
 * lama). Server menolak permintaan lintas-origin, jadi bila origin sendiri melayani API, pakai itu.
 */
export async function reconcileBase() {
  const here = window.location.origin;
  if (connection.baseUrl.replace(/\/+$/, '') === here) return;
  try {
    const res = await fetch(`${here}/api/auth/status`);
    const d = res.ok ? await res.json() : null;
    if (d && typeof d.required === 'boolean') {
      connection.baseUrl = here;
      persist(here, connection.library);
    }
  } catch {
    /* origin ini tidak melayani API: biarkan alamat tersimpan */
  }
}

/** Validasi + simpan koneksi; menyetel folder scan di server bila berbeda. */
export async function saveSetup() {
  setup.error = '';
  const baseUrl = setup.auto ? window.location.origin : connection.baseUrl.trim().replace(/\/+$/, '');
  const library = connection.library.trim();
  if (!baseUrl || !library) {
    setup.error = setup.auto ? 'Folder target scan wajib diisi.' : 'URL proxy/server dan folder target scan wajib diisi.';
    return;
  }
  try {
    new URL(baseUrl);
  } catch {
    setup.error = 'Masukkan URL lengkap, misalnya http://127.0.0.1:5173.';
    return;
  }
  setup.saving = true;
  try {
    // Server dengan token: simpan koneksi lalu minta login (library sudah tersimpan di server).
    await checkAuth();
    if (auth.required && !auth.ok) {
      persist(baseUrl, library);
      connection.library = library;
      setup.configured = true;
      return;
    }
    if (!(await fetch(api('/api/health'))).ok) throw new Error('Server tidak merespons endpoint health.');
    const current = await fetch(api('/api/config'));
    const existing = await current.json().catch(() => ({}));
    if (!current.ok) throw new Error(existing.error || 'Konfigurasi server tidak dapat dibaca.');
    if (existing.library === library) {
      connection.library = existing.library;
    } else {
      const res = await fetch(api('/api/config'), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ library }),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(body.error || 'Target scan ditolak server.');
      connection.library = body.library;
    }
    persist(baseUrl, connection.library);
    setup.configured = true;
    setup.needsLibrary = false;
    await checkHealth();
  } catch (err) {
    setup.error = err.message || 'Koneksi ke server gagal.';
  } finally {
    setup.saving = false;
  }
}

/** Simpan dari halaman Pengaturan (menandai 'tersimpan' bila berhasil). */
export async function saveSettings() {
  setup.saved = false;
  await saveSetup();
  setup.saved = !setup.error;
}
