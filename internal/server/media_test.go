package server

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// Dimensi yang direset scan (width=0) harus pulih walau thumbnail sudah ada di cache.
func TestThumbRestoresDimsFromCache(t *testing.T) {
	s, lib := newTestServer(t)
	t.Cleanup(s.bg.Wait)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 200, 100)), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "real.jpg"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := s.DB.QueryRow(`SELECT id FROM asset WHERE path = 'real.jpg'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	url := "/api/assets/" + itoa(id) + "/thumb"
	dims := func() (w, hh int) {
		if err := s.DB.QueryRow(`SELECT width, height FROM asset WHERE id = ?`, id).Scan(&w, &hh); err != nil {
			t.Fatal(err)
		}
		return
	}

	if rec := do(h, "GET", url, "localhost", nil, ""); rec.Code != 200 {
		t.Fatalf("thumb pertama = %d %s", rec.Code, rec.Body)
	}
	if w, hh := dims(); w <= 0 || hh <= 0 {
		t.Fatalf("dimensi tidak tercatat setelah thumbnail dibuat: %dx%d", w, hh)
	}

	// meniru scan yang melihat mtime berubah, sementara cache thumbnail masih segar
	s.DB.Exec(`UPDATE asset SET width = 0, height = 0 WHERE id = ?`, id)
	if rec := do(h, "GET", url, "localhost", nil, ""); rec.Code != 200 {
		t.Fatalf("thumb dari cache = %d", rec.Code)
	}
	if w, hh := dims(); w <= 0 || hh <= 0 {
		t.Fatalf("dimensi tidak pulih dari cache: %dx%d", w, hh)
	}
}
