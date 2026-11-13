package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// siapkan server dengan 2 aset + album berisi a.jpg saja
func shareFixture(t *testing.T) (*Server, http.Handler, http.Handler, map[string]int64) {
	t.Helper()
	s, _ := newTestServer(t)
	t.Cleanup(s.bg.Wait) // tunggu pemanasan thumbnail sebelum direktori sementara dihapus
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	for _, name := range []string{"a.jpg", "b.mp4"} {
		var id int64
		if err := s.DB.QueryRow(`SELECT id FROM asset WHERE path = ?`, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[name] = id
	}
	res, _ := s.DB.Exec(`INSERT INTO album (name, created_at) VALUES ('Liburan', 1)`)
	alb, _ := res.LastInsertId()
	ids["album"] = alb
	s.DB.Exec(`INSERT INTO album_asset (album_id, asset_id) VALUES (?, ?)`, alb, ids["a.jpg"])
	return s, s.Handler(), s.PublicHandler(), ids
}

func jsonBody(rec *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	json.Unmarshal(rec.Body.Bytes(), &m)
	return m
}

func create(t *testing.T, h http.Handler, body string) map[string]any {
	t.Helper()
	rec := do(h, "POST", "/api/shares", "localhost", nil, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create share = %d %s", rec.Code, rec.Body)
	}
	return jsonBody(rec)
}

func TestShareAlbumFlowAndLeakage(t *testing.T) {
	_, priv, pub, ids := shareFixture(t)
	out := create(t, priv, `{"kind":"album","target_id":`+itoa(ids["album"])+`}`)
	key := out["key"].(string)
	if len(key) != 22 || !strings.HasPrefix(out["path"].(string), "/s/") {
		t.Fatalf("kunci/path tak sesuai: %v", out)
	}

	// pengunjung lewat listener publik, tanpa kredensial apa pun
	meta := jsonBody(do(pub, "GET", "/api/s/"+key, "foto.contoh.id", nil, ""))
	if meta["requires_password"] != false || meta["title"] != "Liburan" || meta["kind"] != "album" {
		t.Fatalf("meta = %v", meta)
	}
	rec := do(pub, "GET", "/api/s/"+key+"/items", "foto.contoh.id", nil, "")
	if rec.Code != 200 {
		t.Fatalf("items = %d %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	for _, bad := range []string{"path", "a.jpg", "b.mp4", "size", "lib"} {
		if strings.Contains(body, `"`+bad+`"`) || strings.Contains(body, bad+".") {
			t.Errorf("items membocorkan %q: %s", bad, body)
		}
	}
	var items struct{ Items []sharedItem }
	json.Unmarshal(rec.Body.Bytes(), &items)
	if len(items.Items) != 1 || items.Items[0].ID != ids["a.jpg"] {
		t.Fatalf("items = %+v, harus hanya a.jpg", items.Items)
	}

	// aset di luar album tidak boleh diakses lewat tautan ini
	base := "/api/s/" + key + "/a/"
	if c := do(pub, "GET", base+itoa(ids["b.mp4"])+"/thumb", "foto.contoh.id", nil, "").Code; c != 404 {
		t.Errorf("aset di luar album = %d, want 404", c)
	}
	// aset di dalam album: preview tersedia, file asli gambar diblokir (unduh tidak diizinkan)
	if c := do(pub, "GET", base+itoa(ids["a.jpg"])+"/original", "foto.contoh.id", nil, "").Code; c != 403 {
		t.Errorf("original tanpa izin unduh = %d, want 403", c)
	}
	// header publik
	rec = do(pub, "GET", "/api/s/"+key, "foto.contoh.id", nil, "")
	if !strings.Contains(rec.Header().Get("X-Robots-Tag"), "noindex") || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("header publik: %v", rec.Header())
	}
}

func TestPublicListenerExposesOnlyShareRoutes(t *testing.T) {
	_, _, pub, _ := shareFixture(t)
	for _, p := range []string{"/api/timeline", "/api/albums", "/api/config", "/api/shares", "/api/assets/1/original",
		"/api/health", "/api/duplicates", "/api/anything"} {
		rec := do(pub, "GET", p, "foto.contoh.id", nil, "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("listener publik GET %s = %d, want 404", p, rec.Code)
		}
	}
	for _, m := range [][2]string{{"POST", "/api/shares"}, {"DELETE", "/api/shares/1"}, {"POST", "/api/scan"}, {"DELETE", "/api/assets/1"}} {
		if c := do(pub, m[0], m[1], "foto.contoh.id", nil, "{}").Code; c != 404 && c != 405 {
			t.Errorf("listener publik %s %s = %d, want 404/405", m[0], m[1], c)
		}
	}
}

func TestSharePasswordAndRateLimit(t *testing.T) {
	_, priv, pub, ids := shareFixture(t)
	key := create(t, priv, `{"kind":"asset","target_id":`+itoa(ids["a.jpg"])+`,"password":"rahasia1"}`)["key"].(string)
	host := "foto.contoh.id"

	meta := jsonBody(do(pub, "GET", "/api/s/"+key, host, nil, ""))
	if meta["requires_password"] != true || meta["unlocked"] != false {
		t.Fatalf("meta = %v", meta)
	}
	if _, has := meta["title"]; has {
		t.Error("judul tidak boleh tampil sebelum dibuka")
	}
	if c := do(pub, "GET", "/api/s/"+key+"/items", host, nil, "").Code; c != 401 {
		t.Errorf("items tanpa unlock = %d, want 401", c)
	}
	if c := do(pub, "GET", "/api/s/"+key+"/a/"+itoa(ids["a.jpg"])+"/thumb", host, nil, "").Code; c != 401 {
		t.Errorf("media tanpa unlock = %d, want 401", c)
	}
	if c := do(pub, "POST", "/api/s/"+key+"/unlock", host, nil, `{"password":"salah"}`).Code; c != 401 {
		t.Errorf("password salah = %d, want 401", c)
	}
	rec := do(pub, "POST", "/api/s/"+key+"/unlock", host, nil, `{"password":"rahasia1"}`)
	if rec.Code != 200 {
		t.Fatalf("unlock = %d %s", rec.Code, rec.Body)
	}
	ck := rec.Result().Cookies()
	if len(ck) != 1 || !ck[0].HttpOnly || !strings.HasPrefix(ck[0].Path, "/api/s/"+key) || strings.Contains(ck[0].Value, "rahasia1") {
		t.Fatalf("cookie unlock tak sesuai: %+v", ck)
	}
	cookie := map[string]string{"Cookie": ck[0].String()}
	if c := do(pub, "GET", "/api/s/"+key+"/items", host, cookie, "").Code; c != 200 {
		t.Errorf("items dengan cookie = %d, want 200", c)
	}
	// cookie dari tautan lain / dirusak tidak berlaku
	bad := map[string]string{"Cookie": ck[0].Name + "=" + strings.Replace(ck[0].Value, ".", ".00", 1)}
	if c := do(pub, "GET", "/api/s/"+key+"/items", host, bad, "").Code; c != 401 {
		t.Errorf("cookie rusak = %d, want 401", c)
	}

	// rate limit percobaan password (6 per 5 menit per IP+tautan); 2 sudah terpakai di atas
	var last int
	for i := 0; i < 8; i++ {
		last = do(pub, "POST", "/api/s/"+key+"/unlock", host, nil, `{"password":"tebak`+itoa(int64(i))+`"}`).Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("setelah banyak percobaan = %d, want 429", last)
	}
}

func TestShareExpiryRevokeAndSoftDelete(t *testing.T) {
	s, priv, pub, ids := shareFixture(t)
	host := "foto.contoh.id"

	// kedaluwarsa
	out := create(t, priv, `{"kind":"asset","target_id":`+itoa(ids["a.jpg"])+`,"expires_in":3600}`)
	key := out["key"].(string)
	if c := do(pub, "GET", "/api/s/"+key, host, nil, "").Code; c != 200 {
		t.Fatalf("sebelum kedaluwarsa = %d", c)
	}
	s.DB.Exec(`UPDATE share SET expires_at = ? WHERE key = ?`, time.Now().Unix()-1, key)
	if c := do(pub, "GET", "/api/s/"+key, host, nil, "").Code; c != 404 {
		t.Errorf("setelah kedaluwarsa = %d, want 404", c)
	}

	// dicabut
	out = create(t, priv, `{"kind":"asset","target_id":`+itoa(ids["a.jpg"])+`}`)
	key2, id2 := out["key"].(string), int64(out["id"].(float64))
	if c := do(priv, "DELETE", "/api/shares/"+itoa(id2), "localhost", nil, "").Code; c != 200 {
		t.Fatalf("revoke = %d", c)
	}
	if c := do(pub, "GET", "/api/s/"+key2, host, nil, "").Code; c != 404 {
		t.Errorf("setelah dicabut = %d, want 404", c)
	}

	// aset dihapus (soft) -> tautan berhenti
	key3 := create(t, priv, `{"kind":"asset","target_id":`+itoa(ids["a.jpg"])+`}`)["key"].(string)
	s.DB.Exec(`UPDATE asset SET deleted_at = 1 WHERE id = ?`, ids["a.jpg"])
	if c := do(pub, "GET", "/api/s/"+key3, host, nil, "").Code; c != 404 {
		t.Errorf("aset dihapus = %d, want 404", c)
	}

	// kunci ngawur / terlalu pendek / mencoba traversal
	for _, k := range []string{"x", strings.Repeat("A", 22), "..%2f..%2fetc", strings.Repeat("a", 64)} {
		if c := do(pub, "GET", "/api/s/"+k, host, nil, "").Code; c != 404 {
			t.Errorf("kunci %q = %d, want 404", k, c)
		}
	}
}

func TestShareDownloadPolicyAndValidation(t *testing.T) {
	_, priv, pub, ids := shareFixture(t)
	host := "foto.contoh.id"
	key := create(t, priv, `{"kind":"asset","target_id":`+itoa(ids["a.jpg"])+`,"allow_download":true}`)["key"].(string)
	rec := do(pub, "GET", "/api/s/"+key+"/a/"+itoa(ids["a.jpg"])+"/original?dl=1", host, nil, "")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Errorf("unduh diizinkan = %d %v", rec.Code, rec.Header())
	}
	// video bisa diputar walau unduhan tidak diizinkan
	vk := create(t, priv, `{"kind":"asset","target_id":`+itoa(ids["b.mp4"])+`}`)["key"].(string)
	if c := do(pub, "GET", "/api/s/"+vk+"/a/"+itoa(ids["b.mp4"])+"/original", host, nil, "").Code; c != 200 {
		t.Errorf("video streaming = %d, want 200", c)
	}

	for name, body := range map[string]string{
		"kind salah":      `{"kind":"x","target_id":1}`,
		"target tak ada":  `{"kind":"asset","target_id":999999}`,
		"expires negatif": `{"kind":"asset","target_id":` + itoa(ids["a.jpg"]) + `,"expires_in":-5}`,
		"expires >1 thn":  `{"kind":"asset","target_id":` + itoa(ids["a.jpg"]) + `,"expires_in":99999999}`,
		"password pendek": `{"kind":"asset","target_id":` + itoa(ids["a.jpg"]) + `,"password":"ab"}`,
	} {
		if c := do(priv, "POST", "/api/shares", "localhost", nil, body).Code; c < 400 {
			t.Errorf("%s = %d, harus ditolak", name, c)
		}
	}
	// daftar milik pemilik memuat kunci; hanya lewat API privat
	rec = do(priv, "GET", "/api/shares", "localhost", nil, "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), key) {
		t.Errorf("list = %d", rec.Code)
	}
}

func TestShareManagementNeedsOwnerToken(t *testing.T) {
	s, priv, pub, ids := shareFixture(t)
	s.Token = "t0ken-pemilik"
	// membuat/melihat/mencabut tautan tetap butuh token pemilik
	for _, c := range [][2]string{{"POST", "/api/shares"}, {"GET", "/api/shares"}, {"DELETE", "/api/shares/1"}} {
		if code := do(priv, c[0], c[1], "localhost", nil, "{}").Code; code != 401 {
			t.Errorf("%s %s tanpa token = %d, want 401", c[0], c[1], code)
		}
	}
	// pengunjung tautan TIDAK butuh token pemilik (di listener publik maupun privat)
	hdr := map[string]string{"Authorization": "Bearer t0ken-pemilik"}
	rec := do(priv, "POST", "/api/shares", "localhost", hdr, `{"kind":"asset","target_id":`+itoa(ids["a.jpg"])+`}`)
	if rec.Code != 201 {
		t.Fatalf("create dgn token = %d", rec.Code)
	}
	key := jsonBody(rec)["key"].(string)
	for _, v := range []struct {
		h    http.Handler
		host string
	}{{pub, "foto.contoh.id"}, {priv, "localhost"}} {
		if c := do(v.h, "GET", "/api/s/"+key, v.host, nil, "").Code; c != 200 {
			t.Errorf("pengunjung tanpa token = %d, want 200", c)
		}
	}
}

func TestPublicHostsAllowlist(t *testing.T) {
	s, _, pub, ids := shareFixture(t)
	_ = ids
	s.PublicHosts = []string{"foto.contoh.id"}
	if c := do(pub, "GET", "/api/s/"+strings.Repeat("a", 22), "foto.contoh.id", nil, "").Code; c != 404 {
		t.Errorf("host terdaftar = %d, want 404 (tautan tak ada, bukan ditolak host)", c)
	}
	if c := do(pub, "GET", "/api/s/"+strings.Repeat("a", 22), "evil.example", nil, "").Code; c != http.StatusMisdirectedRequest {
		t.Errorf("host lain = %d, want 421", c)
	}
}

func TestSharePasswordHash(t *testing.T) {
	enc, err := hashSharePassword("kata-sandi")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(enc, "kata-sandi") || !checkSharePassword(enc, "kata-sandi") || checkSharePassword(enc, "Kata-sandi") {
		t.Fatalf("hash/verify tak benar: %s", enc)
	}
	e2, _ := hashSharePassword("kata-sandi")
	if enc == e2 {
		t.Error("salt harus acak")
	}
}

// JPEG valid dengan segmen komentar berisi penanda "lokasi" (mewakili EXIF/GPS).
func jpegWithSecret(t *testing.T, secret string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := 0; x < 64; x++ {
		for y := 0; y < 48; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 5), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	seg := append([]byte{0xFF, 0xFE, byte((len(secret) + 2) >> 8), byte((len(secret) + 2) & 0xFF)}, secret...)
	return append(append([]byte{b[0], b[1]}, seg...), b[2:]...)
}

func TestSharedDerivativesNeverLeakOriginalMetadata(t *testing.T) {
	s, priv, pub, _ := shareFixture(t)
	const secret = "GPS-SECRET-LAT-0.123"
	if err := os.WriteFile(filepath.Join(s.Library, "gps.jpg"), jpegWithSecret(t, secret), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := s.DB.QueryRow(`SELECT id FROM asset WHERE path = 'gps.jpg'`).Scan(&id); err != nil {
		t.Fatal(err)
	}

	// pemilik: preview JPEG web-safe = file asli (memang begitu, ini area privat)
	own := do(priv, "GET", "/api/assets/"+itoa(id)+"/preview", "localhost", nil, "")
	if !strings.Contains(own.Body.String(), secret) {
		t.Skip("preview pemilik tidak lagi menyajikan file asli; tes pembanding tak relevan")
	}

	key := create(t, priv, `{"kind":"asset","target_id":`+itoa(id)+`}`)["key"].(string)
	for _, what := range []string{"thumb", "preview"} {
		rec := do(pub, "GET", "/api/s/"+key+"/a/"+itoa(id)+"/"+what, "foto.contoh.id", nil, "")
		if rec.Code != 200 {
			t.Fatalf("share %s = %d", what, rec.Code)
		}
		if strings.Contains(rec.Body.String(), secret) {
			t.Errorf("share %s MEMBOCORKAN metadata asli", what)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
			t.Errorf("share %s content-type = %q", what, ct)
		}
	}
	// file asli tetap diblokir tanpa izin unduh
	if c := do(pub, "GET", "/api/s/"+key+"/a/"+itoa(id)+"/original", "foto.contoh.id", nil, "").Code; c != 403 {
		t.Errorf("original = %d, want 403", c)
	}
}

func TestPublicStatusMarker(t *testing.T) {
	_, _, pub, _ := shareFixture(t)
	rec := do(pub, "GET", "/api/auth/status", "foto.contoh.id", nil, "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"public_only":true}` {
		t.Fatalf("status publik = %d %s", rec.Code, rec.Body.String())
	}
}
