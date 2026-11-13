package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/marzukiberg/pigura/internal/db"
	"github.com/marzukiberg/pigura/internal/thumbs"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib")
	if err := os.MkdirAll(filepath.Join(lib, "thumbs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"a.jpg":        "aaaa",
		"b.mp4":        "bbbbbb",
		"note.txt":     "ignored",
		"thumbs/t.jpg": "skipped",
	} {
		if err := os.WriteFile(filepath.Join(lib, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return &Server{DB: d, Library: lib, ScanBase: dir, Gen: thumbs.New(filepath.Join(dir, "cache"), 1)}, lib
}

func TestScanTimelineOriginal(t *testing.T) {
	s, _ := newTestServer(t)
	res, err := s.RunScan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 2 || res.Changed != 2 {
		t.Fatalf("scan = %+v, want seen=2 changed=2", res)
	}
	// scan ulang tanpa perubahan -> tidak ada yang berubah
	if res, _ = s.RunScan(context.Background()); res.Changed != 0 {
		t.Fatalf("rescan changed = %d, want 0", res.Changed)
	}

	h := s.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("GET", "/api/timeline?limit=1", nil))
	var tl struct {
		Items []assetItem `json:"items"`
		Next  *cursor     `json:"next"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tl); err != nil {
		t.Fatal(err)
	}
	if len(tl.Items) != 1 || tl.Next == nil {
		t.Fatalf("timeline = %+v", tl)
	}

	// Filter rentang waktu: from inklusif, to eksklusif.
	taken := tl.Items[0].TakenAt
	for _, c := range []struct {
		q    string
		want int
	}{
		{"from=" + itoa(taken) + "&to=" + itoa(taken+1), 1},
		{"to=" + itoa(taken), 0},
		{"from=" + itoa(taken+1), 0},
	} {
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, newReq("GET", "/api/timeline?limit=500&"+c.q, nil))
		var r struct {
			Items []assetItem `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		got := 0
		for _, it := range r.Items {
			if it.TakenAt != taken {
				t.Fatalf("%s: item di luar rentang: %d", c.q, it.TakenAt)
			}
			got++
		}
		if got < c.want || (c.want == 0 && got != 0) {
			t.Fatalf("%s: got %d item, want %d", c.q, got, c.want)
		}
	}

	// ids= membatasi hasil ke id tertentu (Favorit); ids kosong/tak valid -> kosong.
	for q, want := range map[string]int{
		"ids=" + itoa(tl.Items[0].ID): 1,
		"ids=999999":                  0,
		"ids=abc":                     0,
	} {
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, newReq("GET", "/api/timeline?"+q, nil))
		var r struct {
			Items []assetItem `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if len(r.Items) != want {
			t.Fatalf("%s: got %d items, want %d", q, len(r.Items), want)
		}
	}

	// order=asc: terlama dulu, kursor melanjutkan ke arah yang sama tanpa duplikat.
	var all, asc []assetItem
	for _, q := range []struct {
		url string
		dst *[]assetItem
	}{{"/api/timeline?limit=500", &all}, {"/api/timeline?limit=500&order=asc", &asc}} {
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, newReq("GET", q.url, nil))
		var r struct {
			Items []assetItem `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		*q.dst = r.Items
	}
	if len(asc) != len(all) || len(all) < 2 {
		t.Fatalf("asc=%d desc=%d", len(asc), len(all))
	}
	for i := range all {
		if asc[i].ID != all[len(all)-1-i].ID {
			t.Fatalf("asc[%d]=%d, want kebalikan desc (%d)", i, asc[i].ID, all[len(all)-1-i].ID)
		}
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("GET", "/api/timeline?limit=1&order=asc", nil))
	var p1 struct {
		Items []assetItem `json:"items"`
		Next  *cursor     `json:"next"`
	}
	json.Unmarshal(rec.Body.Bytes(), &p1)
	if p1.Next == nil {
		t.Fatal("asc halaman 1 tanpa kursor")
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("GET", "/api/timeline?limit=1&order=asc&before="+itoa(p1.Next.Before)+"&before_id="+itoa(p1.Next.BeforeID), nil))
	var p2 struct {
		Items []assetItem `json:"items"`
	}
	json.Unmarshal(rec.Body.Bytes(), &p2)
	if len(p2.Items) != 1 || p2.Items[0].ID != asc[1].ID {
		t.Fatalf("asc halaman 2 = %+v, want id %d", p2.Items, asc[1].ID)
	}

	// Deep-link metadata harus dapat dimuat tanpa mengambil timeline lengkap.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("GET", "/api/assets/"+itoa(tl.Items[0].ID), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset detail = %d %s", rec.Code, rec.Body)
	}

	// Range request harus didukung
	req := newReq("GET", "/api/assets/"+itoa(tl.Items[0].ID)+"/original", nil)
	req.Header.Set("Range", "bytes=0-1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want 206", rec.Code)
	}
}

func TestSafeJoin(t *testing.T) {
	if _, err := safeJoin("/lib", "../etc/passwd"); err != nil {
		// path.Clean("/../etc/passwd") => "/etc/passwd" -> tetap di dalam root
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := safeJoin("/lib", "../etc/passwd")
	if got != "/lib/etc/passwd" {
		t.Fatalf("safeJoin = %q, want /lib/etc/passwd", got)
	}
}

func TestAlbums(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := s.Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("POST", "/api/albums", stringsReader(`{"name":"Liburan"}`)))
	if rec.Code != 201 {
		t.Fatalf("create = %d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("POST", "/api/albums/1/assets", stringsReader(`{"ids":[1,2,999]}`)))
	if rec.Code != 200 {
		t.Fatalf("add = %d %s", rec.Code, rec.Body)
	}
	var out map[string]int
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["added"] != 2 {
		t.Fatalf("added = %d, want 2 (id 999 tidak ada)", out["added"])
	}
}

func TestConfigureLibrary(t *testing.T) {
	s, library := newTestServer(t)
	h := s.Handler()

	rec := httptest.NewRecorder()
	req := newReq("PUT", "/api/config", stringsReader(`{"library":"`+library+`"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("configure = %d %s", rec.Code, rec.Body)
	}
	saved, err := db.Setting(s.DB, db.LibraryRootKey)
	if err != nil || saved == "" {
		t.Fatalf("saved library = %q, err=%v", saved, err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("PUT", "/api/config", stringsReader(`{"library":"/"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unsafe configure = %d, want 400", rec.Code)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }

func TestDeleteAssets(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := s.Handler()

	// Single delete
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("DELETE", "/api/assets/1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}

	// Lookup after delete -> 404
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("GET", "/api/assets/1", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("deleted asset status = %d, want 404", rec.Code)
	}

	// Batch delete
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, newReq("POST", "/api/assets/delete", stringsReader(`{"ids":[2]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("batch delete = %d %s", rec.Code, rec.Body)
	}
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["deleted"].(float64) != 1 {
		t.Fatalf("deleted = %v, want 1", out["deleted"])
	}
}

// newReq: seperti httptest.NewRequest tetapi dengan Host loopback (host publik asing ditolak server).
func newReq(method, target string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, target, body)
	r.Host = "localhost:18080"
	return r
}
