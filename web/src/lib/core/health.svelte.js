// Status server (jumlah aset, pemindaian) dan pemicu pindai ulang.
import { api } from './api.svelte.js';

export const health = $state({ ok: true, assets: 0, pending: 0, scanning: false });
export const scanUi = $state({ busy: false });

export async function checkHealth() {
  try {
    const res = await fetch(api('/api/health'));
    if (res.ok) Object.assign(health, await res.json());
  } catch {
    /* offline / sedang start */
  }
}

export async function triggerScan() {
  if (scanUi.busy) return;
  scanUi.busy = true;
  try {
    await fetch(api('/api/scan'), { method: 'POST' });
    await checkHealth();
  } catch {
    /* abaikan */
  } finally {
    setTimeout(() => (scanUi.busy = false), 1500);
  }
}
