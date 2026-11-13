package dupes

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marzukiberg/pigura/internal/db"
	"github.com/marzukiberg/pigura/internal/scanner"
)

func write(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunFindsExactDuplicatesOnly(t *testing.T) {
	root := t.TempDir()
	d, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	rnd := rand.New(rand.NewSource(1))
	big := make([]byte, 400<<10)
	rnd.Read(big)
	// sama ukuran + sama di titik sampel, beda di area yang tidak disampel
	tricky := bytes.Clone(big)
	tricky[100<<10] ^= 0xff
	small := []byte("kecil-sekali")

	write(t, root, "a.jpg", big)
	write(t, root, "b.jpg", big)
	write(t, root, "c.jpg", tricky)
	write(t, root, "s1.png", small)
	write(t, root, "s2.png", small)
	write(t, root, "unik.jpg", []byte("unik"))

	ctx := context.Background()
	if _, err := scanner.Scan(ctx, d, root); err != nil {
		t.Fatal(err)
	}
	var phases []string
	if err := Run(ctx, d, root, 2, func(p Progress) { phases = append(phases, p.Phase) }); err != nil {
		t.Fatal(err)
	}

	sum, err := Summarize(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Groups != 2 || sum.ExtraFiles != 2 {
		t.Fatalf("summary = %+v, want 2 groups / 2 extra files", sum)
	}
	if want := int64(len(big) + len(small)); sum.WastedBytes != want {
		t.Fatalf("wasted = %d, want %d", sum.WastedBytes, want)
	}
	gs, _ := Groups(ctx, d, 10, 0)
	if len(gs) != 2 || gs[0].Size != int64(len(big)) {
		t.Fatalf("groups = %+v (terbesar harus pertama)", gs)
	}

	// file unik tidak boleh pernah di-hash
	var n int
	d.QueryRow(`SELECT COUNT(*) FROM asset WHERE path = 'unik.jpg' AND (qhash IS NOT NULL OR sha1 IS NOT NULL)`).Scan(&n)
	if n != 0 {
		t.Fatal("file berukuran unik seharusnya tidak di-hash")
	}

	// jalankan ulang: semua hash sudah tersimpan -> tidak ada pekerjaan
	phases = nil
	if err := Run(ctx, d, root, 2, func(p Progress) { phases = append(phases, p.Phase) }); err != nil {
		t.Fatal(err)
	}
	if len(phases) != 0 {
		t.Fatalf("run kedua mengerjakan ulang: %v", phases)
	}

	// file berubah -> scan mengosongkan hash -> tidak lagi duplikat
	time.Sleep(1100 * time.Millisecond) // mtime detik
	write(t, root, "b.jpg", append(bytes.Clone(big), 1))
	if _, err := scanner.Scan(ctx, d, root); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, d, root, 2, nil); err != nil {
		t.Fatal(err)
	}
	sum, _ = Summarize(ctx, d)
	if sum.Groups != 1 {
		t.Fatalf("setelah file berubah groups = %d, want 1", sum.Groups)
	}
}

func TestMigrationAddsQhash(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.db")
	d, err := db.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	d.Exec(`ALTER TABLE asset DROP COLUMN qhash`)
	d.Close()
	d, err = db.Open(p) // harus menambahkan kolom lagi tanpa error
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.Exec(`SELECT qhash FROM asset LIMIT 1`); err != nil {
		t.Fatal(err)
	}
}
