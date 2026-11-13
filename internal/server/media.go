// Penyajian file & derivatif (thumbnail/preview/original) dengan perlindungan path.
package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/marzukiberg/pigura/internal/thumbs"
)

// insideLibrary memastikan file (setelah symlink diselesaikan) tetap di dalam library.
func (s *Server) insideLibrary(full string) bool {
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		return false
	}
	root, err := filepath.EvalSymlinks(s.Library)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, real)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// safeJoin menggabungkan root dan rel, menolak path yang keluar dari root.
func safeJoin(root, rel string) (string, error) {
	if root == "" {
		return "", errors.New("root not configured")
	}
	clean := filepath.Join(root, filepath.FromSlash(path.Clean("/"+rel)))
	r, err := filepath.Rel(root, clean)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes root")
	}
	return clean, nil
}

func serveFile(w http.ResponseWriter, r *http.Request, full string) {
	f, err := os.Open(full)
	if err != nil {
		httpErr(w, http.StatusNotFound, "file not found")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		httpErr(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, filepath.Base(full), st.ModTime(), f) // mendukung Range
}

func isWebSafeImage(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg":
		return true
	}
	return false
}

func serveFallbackThumb(w http.ResponseWriter, r *http.Request, kind string) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	var icon string
	if kind == "video" {
		icon = `<path fill="#80868b" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z"/>`
	} else {
		icon = `<path fill="#80868b" d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>`
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 400" width="400" height="400">
		<rect width="400" height="400" fill="#f1f3f4"/>
		<g transform="translate(176, 176) scale(2)">%s</g>
	</svg>`, icon)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}

type assetRow struct {
	ID    int64
	Rel   string
	Type  string
	Mtime int64
	Width int // 0 = dimensi belum tercatat (perlu diisi dari thumbnail); <0 = gagal dibuat
}

func (s *Server) assetFromRequest(r *http.Request) (assetRow, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return assetRow{}, sql.ErrNoRows
	}
	a := assetRow{ID: id}
	err = s.DB.QueryRowContext(r.Context(),
		`SELECT path, type, mtime, width FROM asset WHERE id = ? AND deleted_at IS NULL`, id).
		Scan(&a.Rel, &a.Type, &a.Mtime, &a.Width)
	return a, err
}

func (s *Server) lookup(w http.ResponseWriter, r *http.Request) (assetRow, string, bool) {
	a, err := s.assetFromRequest(r)
	if errors.Is(err, sql.ErrNoRows) {
		httpErr(w, 404, "not found")
		return a, "", false
	} else if err != nil {
		serverErr(w, err)
		return a, "", false
	}
	full, err := safeJoin(s.Library, a.Rel)
	if err != nil {
		httpErr(w, 400, "bad path")
		return a, "", false
	}
	if !s.insideLibrary(full) { // symlink keluar library / file hilang
		httpErr(w, 404, "not found")
		return a, "", false
	}
	return a, full, true
}

func (s *Server) original(w http.ResponseWriter, r *http.Request) {
	if _, full, ok := s.lookup(w, r); ok {
		serveFile(w, r, full)
	}
}

// ensureThumb membuat thumbnail dan, bila perlu, mencatat dimensinya.
func (s *Server) ensureThumb(ctx context.Context, a assetRow, src string, needDims bool) (string, error) {
	if s.Gen == nil {
		return "", thumbs.ErrNoTool
	}
	dst, generated, err := s.Gen.Ensure(ctx, "thumb", a.ID, src, a.Type == "video", a.Mtime, thumbs.ThumbEdge)
	if err != nil {
		if ctx.Err() == nil && !errors.Is(err, thumbs.ErrBusy) { // bukan dibatalkan klien/antrean penuh -> tandai gagal agar tidak diulang terus
			s.DB.ExecContext(context.Background(),
				`UPDATE asset SET width = -1, height = -1 WHERE id = ? AND width = 0`, a.ID)
		}
		return "", err
	}
	if needDims || generated {
		if w, h, derr := thumbs.Dims(dst); derr == nil {
			s.DB.ExecContext(context.Background(),
				`UPDATE asset SET width = ?, height = ? WHERE id = ?`, w, h, a.ID)
		}
	}
	return dst, nil
}

// GET /api/assets/{id}/thumb — JPEG ~400px dari cache (dibuat bila belum ada).
func (s *Server) thumb(w http.ResponseWriter, r *http.Request) {
	a, full, ok := s.lookup(w, r)
	if !ok {
		return
	}
	// Scan mereset dimensi ke 0 saat mtime berubah, tetapi thumbnail di cache bisa masih dianggap segar
	// (mis. mtime mundur): catat dimensinya dari cache, bila tidak layout Grid macet jadi persegi.
	dst, err := s.ensureThumb(r.Context(), a, full, a.Width == 0)
	if errors.Is(err, thumbs.ErrBusy) {
		w.Header().Set("Retry-After", "3")
		httpErr(w, http.StatusServiceUnavailable, "busy, retry shortly")
		return
	}
	if err != nil {
		log.Printf("thumb %d (%s): %v", a.ID, a.Rel, err)
		if a.Type == "image" && isWebSafeImage(a.Rel) {
			serveFile(w, r, full)
			return
		}
		serveFallbackThumb(w, r, a.Type)
		return
	}
	serveFile(w, r, dst)
}

// GET /api/assets/{id}/preview — JPEG ~1920px untuk lightbox (HEIC dkk tidak
// bisa dibuka browser). Video dan kegagalan konversi jatuh ke thumbnail/file aman.
func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	a, full, ok := s.lookup(w, r)
	if !ok {
		return
	}
	if a.Type == "video" {
		serveFile(w, r, full)
		return
	}
	// Bila file asli sudah merupakan gambar web-safe (JPEG/PNG/WebP/GIF),
	// sajikan langsung resolusi asli agar respons instan dan tidak membebani CPU server.
	if isWebSafeImage(a.Rel) {
		serveFile(w, r, full)
		return
	}
	if s.Gen != nil {
		dst, _, err := s.Gen.Ensure(r.Context(), "preview", a.ID, full, false, a.Mtime, thumbs.PreviewEdge)
		if err == nil {
			serveFile(w, r, dst)
			return
		}
		if errors.Is(err, thumbs.ErrBusy) {
			w.Header().Set("Retry-After", "3")
			httpErr(w, http.StatusServiceUnavailable, "busy, retry shortly")
			return
		}
		log.Printf("preview %d (%s): %v — coba fallback thumbnail/asli", a.ID, a.Rel, err)
	}
	// Fallback 1: Gunakan thumbnail yang sudah berhasil diubah jadi JPEG
	if dst, err := s.ensureThumb(r.Context(), a, full, false); err == nil {
		serveFile(w, r, dst)
		return
	}
	// Fallback 2: SVG placeholder elegan agar browser tidak menampilkan broken image
	serveFallbackThumb(w, r, a.Type)
}
