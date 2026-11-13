package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func read(t *testing.T, f fs.FS, name string) string {
	t.Helper()
	b, err := fs.ReadFile(f, name)
	if err != nil {
		return "ERR:" + err.Error()
	}
	return string(b)
}

func TestOverlayPrefersDirAndFallsBack(t *testing.T) {
	base := fstest.MapFS{
		"index.html":    {Data: []byte("TERTANAM")},
		"assets/old.js": {Data: []byte("old")},
	}
	dir := t.TempDir()
	o := Overlay(os.DirFS(dir), base)

	// folder kosong: semuanya dari versi tertanam
	if got := read(t, o, "index.html"); got != "TERTANAM" {
		t.Fatalf("folder kosong = %q", got)
	}

	// deploy frontend: file baru di folder menang, file lain tetap dari cadangan
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("BARU"), 0o644)
	if got := read(t, o, "index.html"); got != "BARU" {
		t.Fatalf("setelah deploy = %q (harus terlihat tanpa restart)", got)
	}
	if got := read(t, o, "assets/old.js"); got != "old" {
		t.Fatalf("cadangan = %q", got)
	}

	// folder diganti saat berjalan (swap)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("BARU-2"), 0o644)
	if got := read(t, o, "index.html"); got != "BARU-2" {
		t.Fatalf("pembaruan = %q", got)
	}

	// folder dihapus: kembali ke versi tertanam, tidak galat
	os.RemoveAll(dir)
	if got := read(t, o, "index.html"); got != "TERTANAM" {
		t.Fatalf("folder hilang = %q", got)
	}
	if _, err := fs.Stat(o, "tidak-ada.js"); err == nil {
		t.Fatal("file yang tak ada di mana pun harus galat")
	}
}
