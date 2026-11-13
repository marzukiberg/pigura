// Package server menyediakan API JSON, penyaji media (HTTP Range) dan SPA.
//
// Berkas per domain: timeline.go, media.go, assets.go, albums.go, library.go, duplicates.go,
// share_*.go (berbagi tautan), security.go + auth.go (lapisan keamanan), spa.go, httpx.go (util HTTP).
package server

import (
	"database/sql"
	"io/fs"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/marzukiberg/pigura/internal/meta"
	"github.com/marzukiberg/pigura/internal/thumbs"
)

// Server menampung dependensi handler.
type Server struct {
	DB       *sql.DB
	Library  string            // root file asli (read-only)
	ScanBase string            // induk aman untuk library yang boleh dipindai
	Gen      *thumbs.Generator // thumbnail & preview (cache di disk)
	Static   fs.FS             // hasil build frontend

	// Keamanan (lihat security.go). Token kosong = tanpa login.
	Token          string
	AllowedHosts   []string // host tambahan di luar loopback/IP/ts.net
	AllowedOrigins []string // origin lintas-situs yang diizinkan (default: tidak ada)

	// Berbagi (lihat share.go)
	PublicURL       string   // mis. https://foto.contoh.id (untuk menyusun tautan); kosong = hanya path
	PublicHosts     []string // host yang diterima listener publik (kosong = tidak dibatasi)
	rl              rateLimiter
	Meta            *meta.Reader // pembaca metadata (nil = vipsheader/ffprobe); diganti di test
	metaCache       metaCache
	bg              sync.WaitGroup // pekerjaan latar belakang (pemanasan thumbnail berbagi); ditunggu di test
	shareSecretOnce sync.Once
	shareSecretVal  []byte

	scanning atomic.Bool

	dupMu     sync.Mutex
	dupStatus dupStatus
}

// Handler membangun router.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.HandleFunc("POST /api/auth/login", s.authLogin)
	mux.HandleFunc("GET /api/config", s.config)
	mux.HandleFunc("PUT /api/config", s.config)
	mux.HandleFunc("GET /api/timeline", s.timeline)
	mux.HandleFunc("GET /api/assets/{id}", s.asset)
	mux.HandleFunc("DELETE /api/assets/{id}", s.deleteAsset)
	mux.HandleFunc("POST /api/assets/delete", s.batchDeleteAssets)
	mux.HandleFunc("GET /api/assets/{id}/thumb", s.thumb)
	mux.HandleFunc("GET /api/assets/{id}/preview", s.preview)
	mux.HandleFunc("GET /api/assets/{id}/original", s.original)
	mux.HandleFunc("GET /api/assets/{id}/metadata", s.assetMetadata)
	mux.HandleFunc("GET /api/albums", s.albums)
	mux.HandleFunc("POST /api/albums", s.createAlbum)
	mux.HandleFunc("PATCH /api/albums/{id}", s.renameAlbum)
	mux.HandleFunc("GET /api/albums/{id}/assets", s.albumAssets)
	mux.HandleFunc("POST /api/albums/{id}/assets", s.addAlbumAssets)
	mux.HandleFunc("POST /api/scan", s.scan)
	mux.HandleFunc("GET /api/duplicates", s.duplicates)
	mux.HandleFunc("POST /api/duplicates/scan", s.duplicatesScan)
	mux.HandleFunc("GET /api/shares", s.listShares)
	mux.HandleFunc("POST /api/shares", s.createShare)
	mux.HandleFunc("DELETE /api/shares/{id}", s.revokeShare)
	s.publicRoutes(mux) // tautan berbagi juga bisa dibuka lewat alamat privat
	mux.Handle("/", s.spa())
	return s.secure(mux)
}
