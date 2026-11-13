# Pigura

**A lightweight, self-hosted photo & video viewer** in the spirit of Google Photos, built to run on tiny
machines (ARM set-top boxes, Raspberry Pi, old laptops, NAS). One static Go binary + SQLite + a Svelte 5 PWA.
No database server, no machine learning, about 20 MB of RAM at idle (see [Performance](#performance)).

*Pigura* is Indonesian for "picture frame". 🇮🇩 [Baca dalam Bahasa Indonesia](README.id.md)

> Pigura is an independent project. It is **not** affiliated with Immich, Google or Google Photos.
> It can auto-detect an existing Immich-style `library/` folder on disk, but it never talks to or modifies Immich.

## Screenshots

<table>
  <tr>
    <td><img src="docs/screenshots/timeline.png" alt="Timeline grouped by day"><br><sub>Timeline, grouped by day</sub></td>
    <td><img src="docs/screenshots/lightbox.png" alt="Photo lightbox"><br><sub>Lightbox with zoom, favorite, download, share</sub></td>
  </tr>
  <tr>
    <td><img src="docs/screenshots/explore.png" alt="Explore view"><br><sub>Explore: filter by type or date</sub></td>
    <td><img src="docs/screenshots/albums.png" alt="Albums"><br><sub>Albums</sub></td>
  </tr>
</table>

<sub>Faces and third-party handles are blurred. The UI is shown in Indonesian.</sub>

## Features

- Timeline with fast keyset pagination, grid / list views, date filter and "explore"
- Albums, favorites, photo lightbox with EXIF info; video playback with HTTP Range
- Thumbnails & previews generated on demand and cached (libvips, libheif for HEIC, ffmpeg for video)
- Incremental library scan + optional periodic auto-scan (great with Syncthing/phone sync)
- Exact duplicate finder (size → sampled hash → full SHA-1)
- Share links with a **separate public listener**, so only the shared items are ever exposed
- Installable PWA with offline thumbnail cache
- Originals are read-only: "delete" only hides a photo inside Pigura, files on disk are never touched
- Hardened by default: host allowlist (anti DNS-rebinding), same-origin only, optional token login,
  CSP and friends, bounded work queues

Status: early (v0.x). The UI is currently in Indonesian; i18n is welcome as a contribution.

> [!WARNING]
> **Security disclaimer.** Pigura is early-stage software (v0.x), provided "as is" under the MIT license, without warranty.
>
> - **There are no user accounts.** Access control is a single, *optional* shared token (`PIGURA_TOKEN`). **If you do not
>   set it, anyone who can reach the port can view, hide and share your photos.** Always set it unless Pigura is only
>   reachable from `127.0.0.1`, for example: `export PIGURA_TOKEN=$(openssl rand -hex 32)`.
>   In `docker-compose.yml` the token line is commented out by default; uncomment it.
> - **Do not expose the main port directly to the internet.** Use a TLS reverse proxy or a private network (VPN,
>   Tailscale, ...). Publish only the separate share-link listener (`PIGURA_PUBLIC_ADDR`) if you need public links.
> - Logins are not individually revocable and there is no logout: the only way to invalidate a session is to change the token.
> - Pigura decodes untrusted media with libvips, libheif and ffmpeg. Keep them updated and run Pigura as an unprivileged
>   user with the library mounted read-only.
> - Pigura never modifies or deletes your original files, but it is **not a backup**. Keep your own backups.

## Quick start (Docker)

```sh
git clone https://github.com/marzukiberg/pigura && cd pigura
# edit docker-compose.yml: set your photos path and PIGURA_TOKEN
docker compose up -d
# open http://127.0.0.1:8080
```

Photos are mounted read-only at `/media/photos`; the index and thumbnail cache live in the `pigura-data` volume.
Images are published for `linux/amd64` and `linux/arm64`.

## Quick start (binary)

Download a release binary (or `make build`), then:

```sh
./pigura -library /srv/photos -scan-base /srv -scan
# open http://127.0.0.1:8080
```

Optional but recommended thumbnail tools: `vipsthumbnail` (libvips), `heif-convert` (libheif), `ffmpeg`.
Without any of them Pigura still works but cannot generate thumbnails.

### Build from source

Requires Go (see `go.mod`) and Node 22+. The frontend is embedded in the binary, so build it first:

```sh
make build          # frontend + binary -> bin/pigura
make arm64          # static Linux ARM64 binary
make test           # go vet + go test
```

A plain `go build` from a fresh clone embeds an empty frontend; always use `make build` (or `make web` first).

## Configuration

Flags take precedence over environment variables.

| Flag | Env | Default | Meaning |
|---|---|---|---|
| `-addr` | `PIGURA_ADDR` | `127.0.0.1:8080` | Listen address |
| `-db` | `PIGURA_DB` | `./data/pigura.db` | SQLite file |
| `-library` | `PIGURA_LIBRARY` | – | Photo root (read-only). Must be a **child** of `-scan-base` |
| `-scan-base` | `PIGURA_SCAN_BASE` | `/srv/photos` | Folders Pigura is allowed to scan; protects against scanning `/` by mistake |
| `-cache` | `PIGURA_CACHE` | `<db dir>/cache` | Thumbnail cache directory |
| `-thumb-workers` | `PIGURA_THUMB_WORKERS` | `1` | Parallel thumbnail processes (keep low on small RAM) |
| `-scan` | – | off | Scan the library at start |
| `-scan-interval` | `PIGURA_SCAN_INTERVAL` | `0` (off) | Periodic scan, e.g. `5m` |
| `-static` | `PIGURA_STATIC_DIR` | – | Serve the frontend from a folder (falls back to the embedded one) |
| – | `PIGURA_TOKEN` | – | If set, login is required (HttpOnly SameSite=Strict cookie or `Authorization: Bearer`) |
| – | `PIGURA_ALLOWED_HOSTS` | – | Extra `Host` names allowed (comma-separated), e.g. your reverse-proxy hostname |
| – | `PIGURA_ALLOWED_ORIGINS` | – | Cross-site origins allowed (comma-separated) |
| – | `PIGURA_PUBLIC_ADDR` | – | Separate listener that serves **only** share links (bind to loopback, point your tunnel/proxy here) |
| – | `PIGURA_PUBLIC_URL` | – | Base URL used to build share links |
| – | `PIGURA_PUBLIC_HOSTS` | – | `Host` names accepted on the public listener |

Tip for low-RAM devices: set `GOMEMLIMIT=80MiB` and `VIPS_CONCURRENCY=2`.

## Performance

Measured on a low-end ARM64 set-top box (Armbian, 4 cores, 1.8 GB RAM shared with other services, spinning HDD)
with a real library of 5,476 items (2,570 HEIC photos, 1,362 videos). Requests went over loopback, so network speed is
excluded. Settings: `PIGURA_THUMB_WORKERS=1` (default), `GOMEMLIMIT=80MiB`, `VIPS_CONCURRENCY=2`.
"Cold" means the thumbnail was not in the cache yet; each thumbnail is generated once and then served from disk.

**Memory**

| Situation | Peak memory |
|---|---|
| Idle server process | 13–18 MB RSS |
| Server process during the whole benchmark (thumbnail jobs run as child processes) | 18 MB RSS |
| Cold JPEG thumbnail (server + libvips) | ~53 MB |
| Cold HEIC thumbnail (server + heif-convert/libvips) | ~121 MB |
| Cold HEVC 1080p video thumbnail (server + ffmpeg) | ~145–150 MB |
| Cold H.264 576p video thumbnail | ~85 MB |
| 8 cold HEIC requests fired at once | ~121 MB (same as a single one) |

**Latency**

| Request | Result |
|---|---|
| Timeline page (~30 KB JSON, 30 requests) | p50 9 ms, p95 10 ms |
| Cached thumbnail (160 requests, 4 concurrent) | p50 5 ms, p95 44 ms, ~60 req/s |
| Cold JPEG thumbnail (2–3 MB source) | ~0.5 s |
| Cold HEIC thumbnail / lightbox preview | ~4.3–4.9 s / ~4.2–4.5 s |
| Cold video thumbnail | 0.9–3.6 s (HEVC 1080p is the slowest) |
| 8 cold HEIC requests at once | processed one at a time, ~4 s each, last one done after ~33 s |

What this means: **serving** photos stays around 20 MB. The short spikes to ~120–150 MB come from the external
decoders and are bounded by `PIGURA_THUMB_WORKERS`, so a screen full of uncached photos queues up instead of using
more memory. The price is that HEIC conversion is slow on weak CPUs; it only happens the first time a photo is shown.

Caveats: a single run on one device, meant as an indication and not a rigorous benchmark. Video files that are
corrupt (e.g. a missing `moov` atom) get a placeholder instead of a thumbnail and are retried on the next request.

## Security model

Pigura has **no user accounts**. Protection is layered (`internal/server/security.go`):

- **Host allowlist** (anti DNS rebinding): loopback, literal IPs, dot-less names, `*.ts.net`, or hosts in `PIGURA_ALLOWED_HOSTS`.
- **Same-origin only**: foreign `Origin` → 403, no CORS headers (unless listed in `PIGURA_ALLOWED_ORIGINS`).
- **Optional token** (`PIGURA_TOKEN`). Unset = no login. **Set it** unless Pigura is only reachable from localhost.
- Security headers (CSP, `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`), `Range` limits,
  bounded thumbnail queue (503 when full), server timeouts.
- Symlinks that escape the library are not served.
- The scanner can only run below `-scan-base`.

Do not expose the main listener directly to the internet; put a TLS reverse proxy in front and only publish the
share-link listener if you need public links. See [SECURITY.md](SECURITY.md) to report vulnerabilities.

## Running as a service

`deploy/pigura.service` is a hardened systemd unit (read-only library, no capabilities, syscall filter, memory cap).

## Project layout

```
cmd/pigura/          entrypoint (flags/env)
internal/db/         SQLite (modernc, no cgo) + schema
internal/scanner/    library walk -> asset table
internal/server/     JSON API: timeline, media (+Range), assets, albums, duplicates, share_*, security/auth, spa
internal/dupes/      exact-duplicate detection
internal/meta/       EXIF/metadata
internal/thumbs/     thumbnails/previews (vips, heif-convert, ffmpeg, pure Go) + bounded queue
internal/web/        embedded frontend (dist/)
web/src/             Svelte 5 + Vite PWA
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Issues and PRs are welcome.

## License

[MIT](LICENSE). Third-party tools invoked as separate processes (libvips, libheif, ffmpeg) keep their own
licenses; the Docker image bundles them from Alpine packages (ffmpeg builds may be GPL).
