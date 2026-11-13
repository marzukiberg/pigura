// Cek duplikat (job latar belakang + hasil).
package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/marzukiberg/pigura/internal/dupes"
)

type dupStatus struct {
	Running    bool   `json:"running"`
	Phase      string `json:"phase"`
	Done       int    `json:"done"`
	Total      int    `json:"total"`
	FinishedAt int64  `json:"finished_at"`
	Error      string `json:"error,omitempty"`
}

const dupWorkers = 2 // I/O-bound di HDD; lebih banyak tidak mempercepat

// POST /api/duplicates/scan — hitung hash kandidat di latar belakang.
func (s *Server) duplicatesScan(w http.ResponseWriter, r *http.Request) {
	if s.Library == "" {
		httpErr(w, http.StatusConflict, "library belum dikonfigurasi")
		return
	}
	s.dupMu.Lock()
	if s.dupStatus.Running {
		s.dupMu.Unlock()
		writeJSON(w, http.StatusAccepted, map[string]bool{"started": false})
		return
	}
	s.dupStatus = dupStatus{Running: true, Phase: "sample"}
	s.dupMu.Unlock()

	go func() {
		start := time.Now()
		err := dupes.Run(context.Background(), s.DB, s.Library, dupWorkers, func(p dupes.Progress) {
			s.dupMu.Lock()
			s.dupStatus.Phase, s.dupStatus.Done, s.dupStatus.Total = p.Phase, p.Done, p.Total
			s.dupMu.Unlock()
		})
		s.dupMu.Lock()
		s.dupStatus.Running = false
		s.dupStatus.FinishedAt = time.Now().Unix()
		if err != nil {
			s.dupStatus.Error = err.Error()
		}
		s.dupMu.Unlock()
		log.Printf("duplicates: selesai dalam %s err=%v", time.Since(start).Round(time.Millisecond), err)
	}()
	writeJSON(w, http.StatusAccepted, map[string]bool{"started": true})
}

// GET /api/duplicates?limit=&offset= — grup duplikat (paling boros ruang dulu).
func (s *Server) duplicates(w http.ResponseWriter, r *http.Request) {
	limit := int(queryInt(r, "limit", 30))
	if limit < 1 || limit > 100 {
		limit = 30
	}
	offset := int(queryInt(r, "offset", 0))
	if offset < 0 {
		offset = 0
	}
	sum, err := dupes.Summarize(r.Context(), s.DB)
	if err != nil {
		serverErr(w, err)
		return
	}
	gs, err := dupes.Groups(r.Context(), s.DB, limit, offset)
	if err != nil {
		serverErr(w, err)
		return
	}
	type outGroup struct {
		dupes.Group
		Items []assetItem `json:"items"`
	}
	out := make([]outGroup, 0, len(gs))
	for _, g := range gs {
		rows, err := s.DB.QueryContext(r.Context(), `
			SELECT id, type, taken_at, width, height, duration, path, size FROM asset
			WHERE deleted_at IS NULL AND sha1 = ? AND size = ?
			ORDER BY taken_at ASC, id ASC`, g.Hash, g.Size)
		if err != nil {
			serverErr(w, err)
			return
		}
		items, err := scanItems(rows)
		if err != nil {
			serverErr(w, err)
			return
		}
		out = append(out, outGroup{Group: g, Items: items})
	}
	s.dupMu.Lock()
	st := s.dupStatus
	s.dupMu.Unlock()
	writeJSON(w, 200, map[string]any{"status": st, "summary": sum, "groups": out})
}
