package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAutoScanIndexesNewFiles(t *testing.T) {
	s, lib := newTestServer(t)
	if _, err := s.RunScan(context.Background()); err != nil {
		t.Fatal(err)
	}
	count := func() (n int) {
		s.DB.QueryRow(`SELECT COUNT(*) FROM asset WHERE deleted_at IS NULL`).Scan(&n)
		return
	}
	before := count()

	// simulasi Syncthing: file baru + folder bantu yang harus diabaikan
	os.WriteFile(filepath.Join(lib, "dari-hp.jpg"), []byte("jpg-baru"), 0o644)
	os.MkdirAll(filepath.Join(lib, ".stversions"), 0o755)
	os.WriteFile(filepath.Join(lib, ".stversions", "versi-lama.jpg"), []byte("lama"), 0o644)
	os.MkdirAll(filepath.Join(lib, ".stfolder"), 0o755)
	os.WriteFile(filepath.Join(lib, ".syncthing.setengah.jpg.tmp"), []byte("tmp"), 0o644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.autoScanLoop(ctx, 20*time.Millisecond) // interval minimum diabaikan untuk loop internal

	deadline := time.Now().Add(3 * time.Second)
	for count() == before && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if got := count(); got != before+1 {
		t.Fatalf("aset = %d, want %d (hanya dari-hp.jpg yang baru)", got, before+1)
	}
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM asset WHERE path LIKE '.stversions/%' OR path LIKE '%.tmp'`).Scan(&n)
	if n != 0 {
		t.Fatalf("file bantu Syncthing ikut terindeks: %d", n)
	}
}

func TestStartAutoScanDisabledAndClamped(t *testing.T) {
	s, _ := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartAutoScan(ctx, 0)  // nonaktif: tidak boleh memulai apa pun
	s.StartAutoScan(ctx, -1) // negatif: nonaktif
	if minAutoScan < 30*time.Second {
		t.Fatal("batas bawah interval terlalu kecil")
	}
}
