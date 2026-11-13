// Preferensi tampilan galeri (disimpan di localStorage, berlaku untuk semua daftar foto).
const KEY = 'pigura-view';

export const OPTIONS = {
  layout: ['justified', 'square', 'list'],
  size: ['sm', 'md', 'lg'],
  type: ['all', 'image', 'video'],
  order: ['desc', 'asc'],
  group: ['day', 'month', 'none'],
};
export const DEFAULTS = { layout: 'justified', size: 'md', type: 'all', order: 'desc', group: 'day' };

function load() {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) || '{}');
    const out = {};
    for (const k of Object.keys(DEFAULTS)) if (OPTIONS[k].includes(v[k])) out[k] = v[k];
    return out;
  } catch {
    return {};
  }
}

export const view = $state({ ...DEFAULTS, ...load() });

function save() {
  try {
    localStorage.setItem(KEY, JSON.stringify($state.snapshot(view)));
  } catch {
    /* private mode */
  }
}

export function setView(key, value) {
  if (!OPTIONS[key]?.includes(value)) return;
  view[key] = value;
  save();
}

export function resetView() {
  Object.assign(view, DEFAULTS);
  save();
}

export const isCustomView = () => Object.keys(DEFAULTS).some((k) => view[k] !== DEFAULTS[k]);
