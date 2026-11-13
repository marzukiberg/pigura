# Contributing to Pigura

Thanks for helping! Pigura aims to stay **small, fast and low-memory** (it targets devices like a 2 GB ARM box), so
changes that add heavy dependencies or background cost need a good reason.

## Dev setup

Requirements: Go (version in `go.mod`), Node 22+. Optional for thumbnails: `vips`, `heif-convert`, `ffmpeg`.

```sh
make web                       # build the frontend once (needed: Go embeds internal/web/dist)
make test                      # go vet + go test
LIBRARY=~/Pictures/some-dir SCAN_BASE=~/Pictures make run   # http://127.0.0.1:8080
cd web && npm run dev          # frontend dev server, proxies /api to :8080 (override with VITE_API_URL)
make check                     # what CI runs (tests + duplication check)
```

## Guidelines

- Keep the Go side dependency-light (pure Go, `CGO_ENABLED=0`).
- Add or update tests for behaviour changes (`internal/*/*_test.go`).
- Security-sensitive areas (`internal/server/security.go`, auth, share links, path handling in `scanner`/`media`)
  deserve extra care; explain the threat you are addressing in the PR.
- Pigura never modifies original files; keep it that way.
- Code comments are currently a mix of Indonesian and English; either is fine for new code.
- Add a one-line entry to `CHANGELOG.md` under `[Unreleased]` for user-visible changes (it is shown in Settings → About).
- Open an issue before large features so we can agree on scope.

By contributing you agree your contributions are licensed under the MIT License.
