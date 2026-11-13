// Rute publik tautan berbagi (/api/s/*) dan listener publik.
package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/marzukiberg/pigura/internal/db"
	"github.com/marzukiberg/pigura/internal/thumbs"
)

func (s *Server) shareSecret() []byte {
	s.shareSecretOnce.Do(func() {
		v, _ := db.Setting(s.DB, "share_secret")
		if len(v) < 32 {
			b := make([]byte, 32)
			rand.Read(b)
			v = hex.EncodeToString(b)
			db.SetSetting(s.DB, "share_secret", v)
		}
		s.shareSecretVal = []byte(v)
	})
	return s.shareSecretVal
}

func (s *Server) unlockValue(sh *shareRow, exp int64) string {
	m := hmac.New(sha256.New, s.shareSecret())
	fmt.Fprintf(m, "%s|%s|%d", sh.Key, sh.PW.String, exp) // pw ikut: ganti password = cookie lama batal
	return fmt.Sprintf("%d.%s", exp, hex.EncodeToString(m.Sum(nil)))
}

func shareCookieName(sh *shareRow) string { return "ils_" + sh.Key[:8] }

func (s *Server) unlocked(r *http.Request, sh *shareRow) bool {
	if !sh.PW.Valid {
		return true
	}
	c, err := r.Cookie(shareCookieName(sh))
	if err != nil {
		return false
	}
	expStr, _, ok := strings.Cut(c.Value, ".")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if !ok || err != nil || exp < time.Now().Unix() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.unlockValue(sh, exp))) == 1
}

func (s *Server) publicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/s/{key}", s.shareMeta)
	mux.HandleFunc("POST /api/s/{key}/unlock", s.shareUnlock)
	mux.HandleFunc("GET /api/s/{key}/items", s.shareItems)
	mux.HandleFunc("GET /api/s/{key}/a/{id}/thumb", s.shareMedia("thumb", s.shareDerived("sthumb", thumbs.ThumbEdge)))
	mux.HandleFunc("GET /api/s/{key}/a/{id}/preview", s.shareMedia("preview", s.shareDerived("spreview", thumbs.PreviewEdge)))
	mux.HandleFunc("GET /api/s/{key}/a/{id}/original", s.shareMedia("original", s.original))
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

func notFoundShare(w http.ResponseWriter) {
	noStore(w)
	httpErr(w, http.StatusNotFound, "not found")
}

// shareFor memuat tautan yang masih berlaku; semua kegagalan menjadi 404 yang sama.
func (s *Server) shareFor(w http.ResponseWriter, r *http.Request) (*shareRow, bool) {
	key := r.PathValue("key")
	if !s.rl.allow("pub|"+clientIP(r), 1500, time.Minute) {
		w.Header().Set("Retry-After", "30")
		httpErr(w, http.StatusTooManyRequests, "too many requests")
		return nil, false
	}
	if !validShareKey(key) {
		notFoundShare(w)
		return nil, false
	}
	sh, err := scanShare(s.DB.QueryRowContext(r.Context(), `SELECT `+shareCols+` FROM share WHERE key = ?`, key))
	if err != nil || sh.status(time.Now().Unix()) != "active" {
		notFoundShare(w)
		return nil, false
	}
	if _, ok := s.targetInfo(sh.Kind, sh.TargetID); !ok {
		notFoundShare(w)
		return nil, false
	}
	return sh, true
}

// GET /api/s/{key} — metadata minimum; judul baru terlihat setelah dibuka.
func (s *Server) shareMeta(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.shareFor(w, r)
	if !ok {
		return
	}
	noStore(w)
	open := s.unlocked(r, sh)
	out := map[string]any{"kind": sh.Kind, "requires_password": sh.PW.Valid, "unlocked": open, "allow_download": sh.AllowDownload}
	if sh.ExpiresAt.Valid {
		out["expires_at"] = sh.ExpiresAt.Int64
	}
	if open {
		out["title"] = sh.Title
		if sh.Kind == "asset" {
			out["title"] = "" // nama file tidak ditampilkan ke pengunjung
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /api/s/{key}/unlock {password}
func (s *Server) shareUnlock(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.shareFor(w, r)
	if !ok {
		return
	}
	noStore(w)
	if !sh.PW.Valid {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if !s.rl.allow("unlock|"+clientIP(r)+"|"+sh.Key, 6, 5*time.Minute) {
		w.Header().Set("Retry-After", "300")
		httpErr(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<10)).Decode(&in); err != nil ||
		!checkSharePassword(sh.PW.String, in.Password) {
		time.Sleep(300 * time.Millisecond)
		httpErr(w, http.StatusUnauthorized, "wrong password")
		return
	}
	exp := time.Now().Unix() + unlockCookieTTL
	if sh.ExpiresAt.Valid && sh.ExpiresAt.Int64 < exp {
		exp = sh.ExpiresAt.Int64
	}
	http.SetCookie(w, &http.Cookie{
		Name: shareCookieName(sh), Value: s.unlockValue(sh, exp), Path: "/api/s/" + sh.Key,
		Expires: time.Unix(exp, 0), HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type sharedItem struct {
	ID       int64   `json:"id"`
	Type     string  `json:"type"`
	TakenAt  int64   `json:"taken_at"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Duration float64 `json:"duration"`
}

// GET /api/s/{key}/items — tanpa path/nama file/ukuran.
func (s *Server) shareItems(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.shareFor(w, r)
	if !ok {
		return
	}
	noStore(w)
	if !s.unlocked(r, sh) {
		httpErr(w, http.StatusUnauthorized, "password required")
		return
	}
	var rows *sql.Rows
	var err error
	if sh.Kind == "album" {
		rows, err = s.DB.QueryContext(r.Context(), `
			SELECT a.id, a.type, a.taken_at, a.width, a.height, a.duration
			FROM album_asset x JOIN asset a ON a.id = x.asset_id
			WHERE x.album_id = ? AND a.deleted_at IS NULL
			ORDER BY a.taken_at DESC, a.id DESC LIMIT ?`, sh.TargetID, maxShareItems)
	} else {
		rows, err = s.DB.QueryContext(r.Context(), `
			SELECT id, type, taken_at, width, height, duration FROM asset WHERE id = ? AND deleted_at IS NULL`, sh.TargetID)
	}
	if err != nil {
		serverErr(w, err)
		return
	}
	defer rows.Close()
	items := []sharedItem{}
	for rows.Next() {
		var it sharedItem
		if err := rows.Scan(&it.ID, &it.Type, &it.TakenAt, &it.Width, &it.Height, &it.Duration); err != nil {
			serverErr(w, err)
			return
		}
		items = append(items, it)
	}
	s.DB.ExecContext(r.Context(), `UPDATE share SET views = views + 1, last_viewed_at = ? WHERE id = ?`, time.Now().Unix(), sh.ID)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "allow_download": sh.AllowDownload})
}

// shareMedia membungkus handler media: pastikan tautan aktif, terbuka, aset termasuk di dalamnya,
// dan kebijakan unduh dipatuhi. Handler asli membaca {id} dari path yang sama.
func (s *Server) shareMedia(what string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sh, ok := s.shareFor(w, r)
		if !ok {
			return
		}
		if !s.unlocked(r, sh) {
			httpErr(w, http.StatusUnauthorized, "password required")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			notFoundShare(w)
			return
		}
		var typ string
		if sh.Kind == "album" {
			err = s.DB.QueryRowContext(r.Context(), `SELECT a.type FROM album_asset x JOIN asset a ON a.id = x.asset_id
				WHERE x.album_id = ? AND a.id = ? AND a.deleted_at IS NULL`, sh.TargetID, id).Scan(&typ)
		} else if id == sh.TargetID {
			err = s.DB.QueryRowContext(r.Context(), `SELECT type FROM asset WHERE id = ? AND deleted_at IS NULL`, id).Scan(&typ)
		} else {
			err = sql.ErrNoRows
		}
		if err != nil {
			notFoundShare(w)
			return
		}
		if what == "original" {
			// Video harus bisa diputar (streaming); file asli gambar hanya bila unduhan diizinkan.
			if typ == "image" && !sh.AllowDownload {
				httpErr(w, http.StatusForbidden, "download not allowed")
				return
			}
			if r.URL.Query().Get("dl") == "1" && sh.AllowDownload {
				var p string
				if s.DB.QueryRowContext(r.Context(), `SELECT path FROM asset WHERE id = ?`, id).Scan(&p) == nil {
					w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(p)))
				}
			}
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		next(w, r)
	}
}

// shareDerived menyajikan thumbnail/preview untuk pengunjung. Selalu derivatif hasil render ulang
// (EXIF/GPS dibuang) dari cache terpisah ("sthumb"/"spreview"), TIDAK PERNAH file asli:
// shortcut preview untuk JPEG/PNG milik pemilik menyajikan file asli beserta lokasinya.
func (s *Server) shareDerived(kind string, edge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, full, ok := s.lookup(w, r)
		if !ok {
			return
		}
		if a.Type == "video" && kind == "spreview" {
			serveFile(w, r, full) // video diputar langsung (streaming)
			return
		}
		if s.Gen == nil {
			serveFallbackThumb(w, r, a.Type)
			return
		}
		dst, _, err := s.Gen.Ensure(r.Context(), kind, a.ID, full, a.Type == "video", a.Mtime, edge)
		switch {
		case errors.Is(err, thumbs.ErrBusy):
			w.Header().Set("Retry-After", "3")
			httpErr(w, http.StatusServiceUnavailable, "busy, retry shortly")
		case err != nil:
			log.Printf("share %s %d: %v", kind, a.ID, err)
			serveFallbackThumb(w, r, a.Type) // placeholder, bukan file asli
		default:
			serveFile(w, r, dst)
		}
	}
}

// PublicHandler: listener untuk internet. HANYA tautan berbagi (/api/s/*) dan UI statis;
// API pemilik (timeline, album, config, hapus, scan, ...) tidak ada di sini sama sekali,
// sehingga tunnel yang diarahkan ke listener ini tidak bisa mengekspos API penuh.
func (s *Server) PublicHandler() http.Handler {
	mux := http.NewServeMux()
	s.publicRoutes(mux)
	// Penanda bagi UI: origin ini hanya melayani tautan berbagi (bukan server pemilik).
	mux.HandleFunc("GET /api/auth/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"public_only": true})
	})
	spa := s.spa()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			notFoundShare(w)
			return
		}
		spa.ServeHTTP(w, r)
	}))
	return s.securePublic(mux)
}
