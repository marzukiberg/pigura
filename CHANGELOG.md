# Changelog

All notable changes to Pigura are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/) (while on 0.x, minor releases may include breaking changes).

This file is also shown inside the app under Settings → About, so keep each entry to a single line.

## [Unreleased]

## [0.2.0] - 2026-11-13

### Added
- Rename albums: right-click an album card (or long-press on touch), or use the pencil button inside an album. Backed by the new `PATCH /api/albums/{id}`.
- Renaming an album also updates the title of its existing share links.
- About section and changelog viewer in Settings.
- Versioning from git tags: `pigura -version`, a `version` field in `/api/health`, and the same version embedded in the binary, Docker image and UI.
- Live demo at demo-pigura.ukay.dev (access code in the README).
- `make cross`, `make build-go`, `make fmt` and `make fmt-check` targets.

### Changed
- Settings uses a two-column layout on wide screens (>=1200px) instead of one long left-aligned column.
- Brand is now written "Pigura" everywhere; the home tab title is just "Pigura".
- The version is no longer stored in `web/package.json`; builds take it from the git tag (`make version` prints it).
- `make web` uses `npm ci` for reproducible frontend builds.
- `.dockerignore` excludes `.env` files from the build context.

### Fixed
- Leaving the Duplicates page threw `toastTimer is not defined` and left the next page stuck on "Memuat foto…".
- Square layout flickering on desktop when the scrollbar appeared and disappeared.
- `package-lock.json` was out of sync, which broke `npm ci` and the Docker build.

## [0.1.0] - 2026-11-13

### Added
- Initial public release: timeline, albums, explore, favorites, duplicates and share links.
- Single static Go binary with SQLite and an embedded Svelte 5 PWA.
- Docker image for linux/amd64 and linux/arm64, with optional vips, heif and ffmpeg thumbnails.
