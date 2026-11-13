// Status muat gambar per-id yang bertahan walau komponen tile di-render ulang (mis. layout berubah).
import { SvelteSet } from 'svelte/reactivity';

export function createThumbState() {
  return { loaded: new SvelteSet(), failed: new SvelteSet() };
}
