// Status server, konfigurasi library, dan pemindaian.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/marzukiberg/pigura/internal/db"
	"github.com/marzukiberg/pigura/internal/scanner"
)

// config menyimpan library target setelah memvalidasinya terhadap ScanBase.
func (s *Server) config(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]string{
			"library":   s.Library,
			"scan_base": s.ScanBase,
		})
		return
	}
	if s.scanning.Load() {
		httpErr(w, http.StatusConflict, "wait for the current scan to finish")
		return
	}
	var in struct {
		Library string `json:"library"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	root, err := scanner.ValidateRoot(strings.TrimSpace(in.Library), s.ScanBase)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := db.SetSetting(s.DB, db.LibraryRootKey, root); err != nil {
		serverErr(w, err)
		return
	}
	s.Library = root
	writeJSON(w, http.StatusOK, map[string]string{"library": root})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	var n, pending int
	if err := s.DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*), COALESCE(SUM(width = 0), 0) FROM asset WHERE deleted_at IS NULL`).
		Scan(&n, &pending); err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"ok": true, "assets": n, "pending": pending,
		"scanning": s.scanning.Load(),
	})
}

// POST /api/scan — jalankan scan di latar belakang (satu per satu).
func (s *Server) scan(w http.ResponseWriter, r *http.Request) {
	if s.Library == "" {
		httpErr(w, 400, "library not configured")
		return
	}
	if !s.scanning.CompareAndSwap(false, true) {
		httpErr(w, http.StatusConflict, "scan already running")
		return
	}
	go func() {
		res, err := scanner.Scan(context.Background(), s.DB, s.Library)
		s.scanning.Store(false)
		log.Printf("scan done: %+v err=%v", res, err)
	}()
	writeJSON(w, http.StatusAccepted, map[string]bool{"started": true})
}

// ErrScanRunning: ada pemindaian lain yang sedang berjalan.
var ErrScanRunning = errors.New("scan already running")

// RunScan dipanggil sinkron (mis. flag -scan saat start).
func (s *Server) RunScan(ctx context.Context) (scanner.Result, error) {
	if !s.scanning.CompareAndSwap(false, true) {
		return scanner.Result{}, ErrScanRunning
	}
	defer s.scanning.Store(false)
	return scanner.Scan(ctx, s.DB, s.Library)
}
