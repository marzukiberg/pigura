package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestRenameAlbum(t *testing.T) {
	_, priv, pub, ids := shareFixture(t)
	alb := itoa(ids["album"])
	out := create(t, priv, `{"kind":"album","target_id":`+alb+`}`)
	key := out["key"].(string)

	rec := do(priv, "PATCH", "/api/albums/"+alb, "localhost", nil, `{"name":"  Pantai 2026  "}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename = %d %s", rec.Code, rec.Body)
	}
	if got := jsonBody(rec)["name"]; got != "Pantai 2026" {
		t.Fatalf("nama tidak di-trim: %v", got)
	}
	if list := do(priv, "GET", "/api/albums", "localhost", nil, "").Body.String(); !strings.Contains(list, "Pantai 2026") || strings.Contains(list, "Liburan") {
		t.Fatalf("daftar album tidak ikut berubah: %s", list)
	}
	// salinan judul pada tautan berbagi harus ikut berubah
	if meta := jsonBody(do(pub, "GET", "/api/s/"+key, "foto.contoh.id", nil, "")); meta["title"] != "Pantai 2026" {
		t.Fatalf("judul share masih lama: %v", meta)
	}
}

func TestRenameAlbumValidation(t *testing.T) {
	_, priv, _, ids := shareFixture(t)
	alb := itoa(ids["album"])
	for name, c := range map[string]struct {
		id, body string
		want     int
	}{
		"kosong":          {alb, `{"name":""}`, 400},
		"spasi saja":      {alb, `{"name":"   "}`, 400},
		"json rusak":      {alb, `{`, 400},
		"terlalu panjang": {alb, `{"name":"` + strings.Repeat("x", maxAlbumName+1) + `"}`, 400},
		"id tak valid":    {"abc", `{"name":"x"}`, 400},
		"tak ditemukan":   {"99999", `{"name":"x"}`, 404},
	} {
		if rec := do(priv, "PATCH", "/api/albums/"+c.id, "localhost", nil, c.body); rec.Code != c.want {
			t.Errorf("%s: kode %d, ingin %d (%s)", name, rec.Code, c.want, rec.Body)
		}
	}
	// tepat di batas panjang masih diterima (dihitung per karakter, bukan byte)
	ok := `{"name":"` + strings.Repeat("é", maxAlbumName) + `"}`
	if rec := do(priv, "PATCH", "/api/albums/"+alb, "localhost", nil, ok); rec.Code != 200 {
		t.Errorf("batas panjang ditolak: %d %s", rec.Code, rec.Body)
	}
}

func TestRenameAlbumBlocksCrossOriginAndNeedsToken(t *testing.T) {
	s, priv, _, ids := shareFixture(t)
	alb := itoa(ids["album"])
	evil := map[string]string{"Origin": "https://evil.example"}
	if rec := do(priv, "PATCH", "/api/albums/"+alb, "localhost", evil, `{"name":"x"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("lintas-origin harus 403, dapat %d", rec.Code)
	}
	s.Token = "rahasia"
	if rec := do(s.Handler(), "PATCH", "/api/albums/"+alb, "localhost", nil, `{"name":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token harus 401, dapat %d", rec.Code)
	}
}
