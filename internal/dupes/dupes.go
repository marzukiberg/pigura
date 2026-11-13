// Package dupes mendeteksi file duplikat persis (isi identik) dengan biaya I/O minimal.
//
// Tiga tahap, masing-masing menyaring kandidat untuk tahap berikutnya:
//  1. ukuran file — hanya SQL, tanpa membaca disk; file dengan ukuran unik tidak pernah dibaca.
//  2. hash sampel (64 KB awal + tengah + akhir) — murah, membuang hampir semua kebetulan ukuran sama.
//  3. SHA-1 penuh — hanya untuk grup yang lolos tahap 2.
//
// Hash disimpan di DB (kolom qhash/sha1) dan dibatalkan otomatis oleh scanner
// saat size/mtime berubah, sehingga menjalankan ulang hampir instan.
package dupes

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

const (
	sample  = 64 << 10 // ukuran tiap potongan sampel
	bufSize = 256 << 10
)

// Progress dilaporkan selama Run.
type Progress struct {
	Phase string `json:"phase"` // "sample" | "full"
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

var bufPool = sync.Pool{New: func() any { b := make([]byte, bufSize); return &b }}

type job struct {
	id   int64
	path string
	size int64
}

type result struct {
	id    int64
	qhash string // tahap sampel
	sha1  string // terisi bila file kecil (sampel = seluruh file) atau tahap penuh
}

// sampleHash membaca potongan awal, tengah, dan akhir (atau seluruh file bila kecil).
func sampleHash(path string, size int64) (q string, full string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	h := sha1.New()
	if size <= 3*sample {
		if _, err := io.Copy(h, f); err != nil {
			return "", "", err
		}
		s := hex.EncodeToString(h.Sum(nil))
		return s, s, nil
	}
	fmt.Fprintf(h, "%d:", size)
	buf := make([]byte, sample)
	for _, off := range []int64{0, size/2 - sample/2, size - sample} {
		n, err := f.ReadAt(buf, off)
		if err != nil && err != io.EOF {
			return "", "", err
		}
		h.Write(buf[:n])
	}
	return hex.EncodeToString(h.Sum(nil)), "", nil
}

func fullHash(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	bp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bp)
	h := sha1.New()
	buf := *bp
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := f.Read(buf)
		h.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func load(ctx context.Context, d *sql.DB, query string) ([]job, error) {
	rows, err := d.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.path, &j.size); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// pool menjalankan fn paralel dan menulis hasil ke DB per batch (satu transaksi).
func pool(ctx context.Context, d *sql.DB, jobs []job, workers int, phase string, report func(Progress),
	fn func(job) result, save string) error {
	if len(jobs) == 0 {
		return nil
	}
	in := make(chan job)
	out := make(chan result, workers*2)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range in {
				if ctx.Err() != nil {
					return
				}
				out <- fn(j)
			}
		}()
	}
	go func() {
		defer close(in)
		for _, j := range jobs {
			select {
			case in <- j:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(out) }()

	var done atomic.Int64
	flush := func(batch []result) error {
		if len(batch) == 0 {
			return nil
		}
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		stmt, err := tx.PrepareContext(ctx, save)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, r := range batch {
			var sh any
			if r.sha1 != "" {
				sh = r.sha1
			}
			if _, err := stmt.ExecContext(ctx, r.qhash, sh, r.id); err != nil {
				return err
			}
		}
		return tx.Commit()
	}
	var batch []result
	for r := range out {
		batch = append(batch, r)
		n := done.Add(1)
		if len(batch) >= 50 {
			if err := flush(batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
		if report != nil && (n%10 == 0 || int(n) == len(jobs)) {
			report(Progress{Phase: phase, Done: int(n), Total: len(jobs)})
		}
	}
	if err := flush(batch); err != nil {
		return err
	}
	return ctx.Err()
}

const sameSize = `size IN (SELECT size FROM asset WHERE deleted_at IS NULL GROUP BY size HAVING COUNT(*) > 1)`

// Run menghitung hash yang belum ada untuk kandidat duplikat.
func Run(ctx context.Context, d *sql.DB, root string, workers int, report func(Progress)) error {
	if workers < 1 {
		workers = 1
	}
	// Tahap 1+2: ukuran sama -> hash sampel.
	jobs, err := load(ctx, d, `SELECT id, path, size FROM asset
		WHERE deleted_at IS NULL AND qhash IS NULL AND size > 0 AND `+sameSize)
	if err != nil {
		return err
	}
	for i := range jobs {
		jobs[i].path = filepath.Join(root, filepath.FromSlash(jobs[i].path))
	}
	err = pool(ctx, d, jobs, workers, "sample", report, func(j job) result {
		q, full, err := sampleHash(j.path, j.size)
		if err != nil { // file hilang/tak terbaca: beri nilai unik agar tak pernah dianggap duplikat
			return result{id: j.id, qhash: fmt.Sprintf("x:%d", j.id)}
		}
		return result{id: j.id, qhash: q, sha1: full}
	}, `UPDATE asset SET qhash = ?, sha1 = COALESCE(?, sha1) WHERE id = ?`)
	if err != nil {
		return err
	}

	// Tahap 3: (size, qhash) sama -> hash penuh.
	jobs, err = load(ctx, d, `SELECT id, path, size FROM asset
		WHERE deleted_at IS NULL AND sha1 IS NULL AND qhash IS NOT NULL AND qhash NOT LIKE 'x:%'
		  AND (size, qhash) IN (SELECT size, qhash FROM asset
		       WHERE deleted_at IS NULL AND qhash IS NOT NULL GROUP BY size, qhash HAVING COUNT(*) > 1)`)
	if err != nil {
		return err
	}
	for i := range jobs {
		jobs[i].path = filepath.Join(root, filepath.FromSlash(jobs[i].path))
	}
	return pool(ctx, d, jobs, workers, "full", report, func(j job) result {
		s, err := fullHash(ctx, j.path)
		if err != nil {
			return result{id: j.id, qhash: fmt.Sprintf("x:%d", j.id)}
		}
		return result{id: j.id, qhash: "", sha1: s}
	}, `UPDATE asset SET qhash = COALESCE(NULLIF(?, ''), qhash), sha1 = COALESCE(?, sha1) WHERE id = ?`)
}

// Group satu kumpulan file berisi identik.
type Group struct {
	Hash  string  `json:"hash"`
	Size  int64   `json:"size"`
	Count int     `json:"count"`
	IDs   []int64 `json:"-"`
}

// Summary ringkasan seluruh duplikat.
type Summary struct {
	Groups      int   `json:"groups"`
	ExtraFiles  int   `json:"extra_files"`
	WastedBytes int64 `json:"wasted_bytes"`
}

const groupBy = `FROM asset WHERE deleted_at IS NULL AND sha1 IS NOT NULL GROUP BY sha1, size HAVING COUNT(*) > 1`

func Summarize(ctx context.Context, d *sql.DB) (Summary, error) {
	var s Summary
	err := d.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(c - 1), 0), COALESCE(SUM(sz * (c - 1)), 0)
		FROM (SELECT COUNT(*) c, size sz `+groupBy+`)`).Scan(&s.Groups, &s.ExtraFiles, &s.WastedBytes)
	return s, err
}

// Groups mengembalikan grup terurut dari yang paling boros ruang.
func Groups(ctx context.Context, d *sql.DB, limit, offset int) ([]Group, error) {
	rows, err := d.QueryContext(ctx, `SELECT sha1, size, COUNT(*) c `+groupBy+`
		ORDER BY size * (c - 1) DESC, sha1 LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	var gs []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.Hash, &g.Size, &g.Count); err != nil {
			rows.Close()
			return nil, err
		}
		gs = append(gs, g)
	}
	rows.Close()
	return gs, rows.Err()
}
