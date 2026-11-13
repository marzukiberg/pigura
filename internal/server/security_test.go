package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func do(h http.Handler, method, target, host string, hdr map[string]string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Host = host
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestHostAllowlistBlocksRebinding(t *testing.T) {
	s, _ := newTestServer(t)
	h := s.Handler()
	for host, want := range map[string]int{
		"localhost:18080":        200,
		"127.0.0.1:18080":        200,
		"100.64.0.10:18080":      200,
		"[fd7a:115c:a1e0::1]":    200,
		"myhost.tail1234.ts.net": 200,
		"myhost:18080":           200,
		"rebind.evil.example":    http.StatusMisdirectedRequest,
		"evil.example:18080":     http.StatusMisdirectedRequest,
	} {
		if got := do(h, "GET", "/api/health", host, nil, "").Code; got != want {
			t.Errorf("Host %q = %d, want %d", host, got, want)
		}
	}
	s.AllowedHosts = []string{"foto.contoh.id"}
	if got := do(h, "GET", "/api/health", "foto.contoh.id", nil, "").Code; got != 200 {
		t.Errorf("host terdaftar = %d, want 200", got)
	}
}

func TestCrossOriginBlocked(t *testing.T) {
	s, _ := newTestServer(t)
	h := s.Handler()
	evil := map[string]string{"Origin": "https://evil.example"}

	for _, c := range []struct{ method, target string }{
		{"GET", "/api/config"}, {"POST", "/api/scan"}, {"DELETE", "/api/assets/1"}, {"PUT", "/api/config"},
	} {
		rec := do(h, c.method, c.target, "localhost:18080", evil, "{}")
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s dari origin asing = %d, want 403", c.method, c.target, rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("%s %s membocorkan ACAO", c.method, c.target)
		}
	}
	if rec := do(h, "OPTIONS", "/api/assets/1", "localhost:18080", evil, ""); rec.Code != http.StatusForbidden {
		t.Errorf("preflight asing = %d, want 403", rec.Code)
	}
	if rec := do(h, "GET", "/api/health", "localhost:18080", map[string]string{"Origin": "null"}, ""); rec.Code != http.StatusForbidden {
		t.Errorf("Origin null = %d, want 403", rec.Code)
	}
	// same-origin tetap jalan
	if rec := do(h, "GET", "/api/health", "localhost:18080", map[string]string{"Origin": "http://localhost:18080"}, ""); rec.Code != 200 {
		t.Errorf("same-origin = %d, want 200", rec.Code)
	}

	// origin yang diizinkan eksplisit mendapat CORS (dengan kredensial), yang lain tetap ditolak
	s.AllowedOrigins = []string{"https://app.contoh.id"}
	ok := do(h, "OPTIONS", "/api/health", "localhost:18080", map[string]string{"Origin": "https://app.contoh.id"}, "")
	if ok.Code != http.StatusNoContent || ok.Header().Get("Access-Control-Allow-Origin") != "https://app.contoh.id" {
		t.Errorf("preflight origin terdaftar = %d %v", ok.Code, ok.Header())
	}
	if rec := do(h, "GET", "/api/health", "localhost:18080", evil, ""); rec.Code != http.StatusForbidden {
		t.Errorf("origin tak terdaftar = %d, want 403", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	s, _ := newTestServer(t)
	rec := do(s.Handler(), "GET", "/api/health", "localhost", nil, "")
	for k, sub := range map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Content-Security-Policy":      "frame-ancestors 'none'",
		"Cross-Origin-Resource-Policy": "same-origin",
	} {
		if !strings.Contains(rec.Header().Get(k), sub) {
			t.Errorf("header %s = %q, want mengandung %q", k, rec.Header().Get(k), sub)
		}
	}
}

func TestRangeAbuseRejected(t *testing.T) {
	s, _ := newTestServer(t)
	rec := do(s.Handler(), "GET", "/api/health", "localhost", map[string]string{"Range": "bytes=" + strings.Repeat("0-1,", 50) + "0-1"}, "")
	if rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("Range berlebihan = %d, want 416", rec.Code)
	}
}

func TestTokenAuth(t *testing.T) {
	s, _ := newTestServer(t)
	h := s.Handler()

	// tanpa token dikonfigurasi: terbuka
	var st map[string]bool
	json.Unmarshal(do(h, "GET", "/api/auth/status", "localhost", nil, "").Body.Bytes(), &st)
	if st["required"] || !st["ok"] {
		t.Fatalf("tanpa token: %v", st)
	}

	s.Token = "rahasia-panjang-123"
	if rec := do(h, "GET", "/api/timeline", "localhost", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa kredensial = %d, want 401", rec.Code)
	}
	if rec := do(h, "GET", "/api/assets/1/thumb", "localhost", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("media tanpa kredensial = %d, want 401", rec.Code)
	}
	if rec := do(h, "GET", "/", "localhost", nil, ""); rec.Code == http.StatusUnauthorized {
		t.Fatal("halaman statis (UI login) tidak boleh diblokir")
	}
	json.Unmarshal(do(h, "GET", "/api/auth/status", "localhost", nil, "").Body.Bytes(), &st)
	if !st["required"] || st["ok"] {
		t.Fatalf("status tanpa login: %v", st)
	}

	if rec := do(h, "POST", "/api/auth/login", "localhost", nil, `{"token":"salah"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token salah = %d, want 401", rec.Code)
	}
	rec := do(h, "POST", "/api/auth/login", "localhost", nil, `{"token":"rahasia-panjang-123"}`)
	if rec.Code != 200 {
		t.Fatalf("login = %d %s", rec.Code, rec.Body)
	}
	ck := rec.Result().Cookies()
	if len(ck) != 1 || !ck[0].HttpOnly || ck[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie sesi = %+v (harus HttpOnly + SameSite=Strict)", ck)
	}
	if strings.Contains(ck[0].Value, "rahasia") {
		t.Fatal("cookie tidak boleh memuat token mentah")
	}
	if rec := do(h, "GET", "/api/timeline", "localhost", map[string]string{"Cookie": ck[0].String()}, ""); rec.Code != 200 {
		t.Fatalf("dengan cookie = %d, want 200", rec.Code)
	}
	if rec := do(h, "GET", "/api/timeline", "localhost", map[string]string{"Authorization": "Bearer rahasia-panjang-123"}, ""); rec.Code != 200 {
		t.Fatalf("dengan Bearer = %d, want 200", rec.Code)
	}
	if rec := do(h, "GET", "/api/timeline", "localhost", map[string]string{"Authorization": "Bearer salah"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Bearer salah = %d, want 401", rec.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	s, _ := newTestServer(t)
	s.Token = "t"
	// isi limiter langsung agar test tidak menunggu penundaan per percobaan
	for i := 0; i < 8; i++ {
		s.rl.record("login")
	}
	if rec := do(s.Handler(), "POST", "/api/auth/login", "localhost", nil, `{"token":"t"}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("setelah 8 kegagalan = %d, want 429 (walau token benar)", rec.Code)
	}
}

func TestSymlinkOutsideLibraryNotServed(t *testing.T) {
	s, lib := newTestServer(t)
	outside := filepath.Join(t.TempDir(), "rahasia.jpg")
	if err := os.WriteFile(outside, []byte("TOP-SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(lib, "link.jpg")); err != nil {
		t.Skip("symlink tidak didukung:", err)
	}
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := s.DB.QueryRow(`SELECT id FROM asset WHERE path = 'link.jpg'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	for _, ep := range []string{"original", "thumb", "preview"} {
		rec := do(h, "GET", "/api/assets/"+itoa(id)+"/"+ep, "localhost", nil, "")
		if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "TOP-SECRET") {
			t.Errorf("%s via symlink = %d (harus 404 tanpa isi file)", ep, rec.Code)
		}
	}
	// file biasa di dalam library tetap dilayani
	var okID int64
	s.DB.QueryRow(`SELECT id FROM asset WHERE path = 'a.jpg'`).Scan(&okID)
	if rec := do(h, "GET", "/api/assets/"+itoa(okID)+"/original", "localhost", nil, ""); rec.Code != 200 {
		t.Errorf("file normal = %d, want 200", rec.Code)
	}
}
