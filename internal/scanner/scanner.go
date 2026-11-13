// Package scanner mengindeks file media di library ke SQLite.
//
// Saat ini hanya path/ukuran/mtime (taken_at = mtime). EXIF, SHA1, pHash
// dan thumbnail menyusul di fase berikutnya.
package scanner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var imageExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".heic": true, ".heif": true, ".avif": true, ".bmp": true,
}

var videoExt = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".webm": true,
	".m4v": true, ".avi": true, ".3gp": true,
}

// Direktori turunan Immich / internal yang dilewati agar thumbnail dan
// hasil encode tidak ikut terindeks sebagai aset.
var skipDirs = map[string]bool{
	"thumbs": true, "encoded-video": true, "profile": true,
	"backups": true, ".trash": true,
	// Folder bantu Syncthing: versi lama file (.stversions) bukan aset, penanda folder (.stfolder) kosong.
	".stversions": true, ".stfolder": true,
}

// Result ringkasan satu kali scan.
type Result struct {
	Seen    int `json:"seen"`
	Changed int `json:"changed"`
}

// ValidateRoot memastikan pemindaian hanya berlangsung pada subfolder dari
// allowedBase. Ini mencegah konfigurasi keliru seperti scan /.
func ValidateRoot(root, allowedBase string) (string, error) {
	if root == "" {
		return "", errors.New("library root is required")
	}
	base, err := filepath.Abs(allowedBase)
	if err != nil {
		return "", fmt.Errorf("resolve scan base: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve library root: %w", err)
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return "", fmt.Errorf("resolve scan base symlink: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve library root symlink: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat library root: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("library root is not a directory")
	}
	rel, err := filepath.Rel(base, root)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("library root %q must be a child of scan base %q", root, base)
	}
	return root, nil
}

// FindImmichLibrary memilih layout Immich lazim di bawah base. Pencarian
// hanya memeriksa kandidat direktori, bukan menelusuri atau memindai media.
func FindImmichLibrary(base string) (string, error) {
	base, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve scan base: %w", err)
	}
	candidates := []string{filepath.Join(base, "immich", "library")}
	if matches, _ := filepath.Glob(filepath.Join(base, "*", "library")); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}
	for _, candidate := range candidates {
		root, err := ValidateRoot(candidate, base)
		if err == nil {
			return root, nil
		}
	}
	return "", fmt.Errorf("Immich library not found below %q", base)
}

// Kind mengembalikan "image", "video", atau "" bila bukan media.
func Kind(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case imageExt[ext]:
		return "image"
	case videoExt[ext]:
		return "video"
	}
	return ""
}

const upsert = `
INSERT INTO asset (path, type, taken_at, size, mtime)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
  size = excluded.size, mtime = excluded.mtime, taken_at = excluded.taken_at,
  width = 0, height = 0, -- 0 = thumbnail/dimensi perlu dibuat ulang
  sha1 = NULL, qhash = NULL -- isi berubah: hash duplikat harus dihitung ulang
WHERE asset.size != excluded.size OR asset.mtime != excluded.mtime`

// Scan menelusuri root dan meng-upsert aset. Hanya baris yang berubah ditulis.
func Scan(ctx context.Context, d *sql.DB, root string) (Result, error) {
	var res Result
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, upsert)
	if err != nil {
		return res, err
	}
	defer stmt.Close()

	err = filepath.WalkDir(root, func(p string, e fs.DirEntry, werr error) error {
		if werr != nil {
			return nil // lewati yang tidak terbaca
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.IsDir() {
			if p != root && skipDirs[e.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		kind := Kind(e.Name())
		if kind == "" {
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		res.Seen++
		mt := info.ModTime().Unix()
		r, err := stmt.ExecContext(ctx, filepath.ToSlash(rel), kind, mt, info.Size(), mt)
		if err != nil {
			return err
		}
		if c, _ := r.RowsAffected(); c > 0 {
			res.Changed++
		}
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, tx.Commit()
}
