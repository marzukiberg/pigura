// URL API dipusatkan agar frontend lokal dapat memakai Vite reverse proxy
// maupun server pigura yang diakses langsung lewat jaringannya.
export const connection = $state({
  baseUrl: typeof window === 'undefined' ? '' : window.location.origin,
  library: '',
});

export function api(path) {
  return `${connection.baseUrl.replace(/\/+$/, '')}${path}`;
}
