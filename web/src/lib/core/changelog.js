// Versi aplikasi (dari web/package.json lewat vite.config.js) dan riwayat perubahan
// yang diurai dari CHANGELOG.md di root repo (satu sumber untuk repo dan UI).
import raw from '../../../../CHANGELOG.md?raw';

export const APP_VERSION = __APP_VERSION__;
export const REPO_URL = 'https://github.com/marzukiberg/pigura';

// Penanda markdown sederhana dibuang agar bisa ditampilkan sebagai teks biasa.
const plain = (s) => s.replace(/`([^`]*)`/g, '$1').replace(/\*\*([^*]*)\*\*/g, '$1').replace(/\[([^\]]*)\]\([^)]*\)/g, '$1');

/**
 * @returns {{version: string, date: string, groups: {title: string, items: string[]}[]}[]}
 */
export function parseChangelog(md) {
  const releases = [];
  let rel = null;
  let group = null;
  for (const line of md.split('\n')) {
    let m;
    if ((m = line.match(/^## \[([^\]]+)\](?:\s*-\s*(\S+))?/))) {
      rel = { version: m[1], date: m[2] || '', groups: [] };
      releases.push(rel);
      group = null;
    } else if (rel && (m = line.match(/^### (.+)/))) {
      group = { title: m[1].trim(), items: [] };
      rel.groups.push(group);
    } else if (group && (m = line.match(/^- (.+)/))) {
      group.items.push(plain(m[1].trim()));
    }
  }
  return releases.filter((r) => r.groups.some((g) => g.items.length));
}

export const changelog = parseChangelog(raw);
