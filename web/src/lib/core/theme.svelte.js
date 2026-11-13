// Tema: 'system' | 'light' | 'dark' (disimpan di localStorage, diterapkan lewat atribut data-theme).
export const themeState = $state({ mode: 'system' });

const root = () => document.documentElement;

export function initTheme() {
  const saved = localStorage.getItem('pigura-theme');
  if (saved) {
    themeState.mode = saved;
    root().setAttribute('data-theme', saved);
  }
}

export function setTheme(mode) {
  themeState.mode = mode;
  if (mode === 'system') {
    root().removeAttribute('data-theme');
    localStorage.removeItem('pigura-theme');
  } else {
    root().setAttribute('data-theme', mode);
    localStorage.setItem('pigura-theme', mode);
  }
}

/** Balik terang/gelap berdasarkan tampilan yang sedang efektif. */
export function toggleTheme() {
  const attr = root().getAttribute('data-theme');
  const dark = attr === 'dark' || (!attr && window.matchMedia('(prefers-color-scheme: dark)').matches);
  setTheme(dark ? 'light' : 'dark');
}
