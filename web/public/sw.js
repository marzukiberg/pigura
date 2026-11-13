// Service worker minimal: cache-first untuk thumbnail, network untuk sisanya.
const THUMBS = 'thumbs-v2';
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== THUMBS).map((k) => caches.delete(k)))
    ).then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (e) => {
  let url;
  try {
    url = new URL(e.request.url);
  } catch {
    return;
  }
  if (e.request.method !== 'GET' || !/^\/api\/assets\/\d+\/thumb$/.test(url.pathname)) return;
  e.respondWith(
    caches.open(THUMBS).then(async (c) => {
      const hit = await c.match(e.request);
      if (hit) {
        const ct = hit.headers.get('content-type') || '';
        if (!ct.includes('svg')) return hit;
        await c.delete(e.request);
      }
      const res = await fetch(e.request);
      const ct = res.headers.get('content-type') || '';
      if (res.ok && !ct.includes('svg')) {
        c.put(e.request, res.clone());
      }
      return res;
    }),
  );
});
