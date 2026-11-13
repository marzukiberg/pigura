package web

import "io/fs"

// overlay menyajikan file dari dir bila ada; bila tidak, dari base (frontend tertanam di binary).
// Pengecekan terjadi tiap Open, jadi folder boleh muncul/berganti saat server berjalan
// (deploy frontend tanpa restart) dan folder yang belum ada otomatis jatuh ke versi tertanam.
type overlay struct{ dir, base fs.FS }

func (o overlay) Open(name string) (fs.File, error) {
	if f, err := o.dir.Open(name); err == nil {
		return f, nil
	}
	return o.base.Open(name)
}

// Overlay menggabungkan dir (prioritas) dengan base (cadangan).
func Overlay(dir, base fs.FS) fs.FS { return overlay{dir: dir, base: base} }
