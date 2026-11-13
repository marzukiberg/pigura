// Seleksi foto di Timeline: kotak centang, tekan lama, dan mode pilih dari FAB.
import { SvelteSet } from 'svelte/reactivity';

const LONG_PRESS_MS = 400;

export class Selection {
  ids = new SvelteSet();
  /** Mode pilih dari FAB: kotak centang tampil tanpa harus tekan lama. */
  mode = $state(false);

  #timer;
  #longPressed = false;
  #onOpen;

  /** @param {(index: number) => void} onOpen dipanggil saat tile diketuk di luar mode pilih */
  constructor(onOpen) {
    this.#onOpen = onOpen;
  }

  get size() {
    return this.ids.size;
  }
  get active() {
    return this.ids.size > 0 || this.mode;
  }
  has = (id) => this.ids.has(id);
  list = () => [...this.ids];

  toggle = (id) => {
    if (this.ids.has(id)) this.ids.delete(id);
    else this.ids.add(id);
  };

  /** Keluar dari seleksi sepenuhnya (kosongkan + matikan mode pilih). */
  exit = () => {
    this.ids.clear();
    this.mode = false;
  };

  /** Pilih/batal pilih semua item satu kelompok. */
  toggleGroup = (group) => {
    if (group.allSelected) group.list.forEach(({ a }) => this.ids.delete(a.id));
    else group.list.forEach(({ a }) => this.ids.add(a.id));
  };

  press = (id) => {
    this.#longPressed = false;
    this.#timer = setTimeout(() => {
      this.#longPressed = true;
      this.toggle(id);
    }, LONG_PRESS_MS);
  };
  release = () => clearTimeout(this.#timer);

  tap = (index, id) => {
    if (this.#longPressed) return (this.#longPressed = false);
    if (this.active) this.toggle(id);
    else this.#onOpen(index);
  };
}
