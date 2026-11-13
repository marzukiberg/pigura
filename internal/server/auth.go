// Token akses opsional: cookie sesi HttpOnly / Bearer, login, status.
package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookie = "pigura_session"
	loginMaxFails = 8 // kegagalan login per menit (global; cukup untuk satu pemilik)
)

func authExempt(r *http.Request) bool {
	// /api/s/* = tautan berbagi: dilindungi kunci acak (+ password), bukan token pemilik.
	return r.URL.Path == "/api/auth/status" || r.URL.Path == "/api/auth/login" ||
		strings.HasPrefix(r.URL.Path, "/api/s/")
}

func (s *Server) sessionValue() string {
	m := hmac.New(sha256.New, []byte(s.Token))
	m.Write([]byte("pigura-session-v1"))
	return hex.EncodeToString(m.Sum(nil))
}

func (s *Server) authed(r *http.Request) bool {
	if s.Token == "" {
		return true
	}
	if c, err := r.Cookie(sessionCookie); err == nil &&
		subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.sessionValue())) == 1 {
		return true
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") &&
		subtle.ConstantTimeCompare([]byte(h[7:]), []byte(s.Token)) == 1 {
		return true
	}
	return false
}

// GET /api/auth/status — apakah token diwajibkan dan apakah sesi ini sudah masuk.
func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"required": s.Token != "", "ok": s.authed(r)})
}

// POST /api/auth/login {token} -> cookie sesi HttpOnly.
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if s.Token == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if s.rl.blocked("login", loginMaxFails, time.Minute) {
		w.Header().Set("Retry-After", "60")
		httpErr(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil ||
		subtle.ConstantTimeCompare([]byte(in.Token), []byte(s.Token)) != 1 {
		s.rl.record("login")
		time.Sleep(400 * time.Millisecond)
		httpErr(w, http.StatusUnauthorized, "invalid token")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: s.sessionValue(), Path: "/",
		MaxAge: 180 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
