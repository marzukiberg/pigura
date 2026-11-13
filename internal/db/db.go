// Package db membuka SQLite (pure Go, tanpa cgo) dan menerapkan skema.
package db

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Open membuka (dan membuat bila perlu) database di path dengan WAL.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(4) // hemat memori di perangkat kecil
	// Migrasi DB lama: kolom qhash ditambahkan sebelum schema (yang mengindeks size).
	var hasAsset int
	_ = d.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='asset'`).Scan(&hasAsset)
	if hasAsset > 0 {
		var hasQ int
		_ = d.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('asset') WHERE name='qhash'`).Scan(&hasQ)
		if hasQ == 0 {
			if _, err := d.Exec(`ALTER TABLE asset ADD COLUMN qhash TEXT`); err != nil {
				d.Close()
				return nil, fmt.Errorf("migrate qhash: %w", err)
			}
		}
	}
	if _, err := d.Exec(schema); err != nil {
		d.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return d, nil
}

const LibraryRootKey = "library_root"

// Setting mengambil nilai konfigurasi yang disimpan. Nilai kosong berarti
// pengaturan belum pernah dibuat.
func Setting(d *sql.DB, key string) (string, error) {
	var value string
	err := d.QueryRow(`SELECT value FROM setting WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SetSetting menyimpan konfigurasi secara atomik.
func SetSetting(d *sql.DB, key, value string) error {
	_, err := d.Exec(`INSERT INTO setting (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
