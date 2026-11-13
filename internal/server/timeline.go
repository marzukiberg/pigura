// Timeline & metadata aset (keyset pagination).
package server

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// width/height: dimensi thumbnail (cukup untuk rasio aspek). 0 = belum
// diproses, -1 = gagal diproses.
type assetItem struct {
	ID       int64   `json:"id"`
	Type     string  `json:"type"`
	TakenAt  int64   `json:"taken_at"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Duration float64 `json:"duration"`
	Path     string  `json:"path"`
	Size     int64   `json:"size"`
}

type cursor struct {
	Before   int64 `json:"before"`
	BeforeID int64 `json:"before_id"`
}

func scanItems(rows *sql.Rows) ([]assetItem, error) {
	defer rows.Close()
	items := []assetItem{}
	for rows.Next() {
		var a assetItem
		if err := rows.Scan(&a.ID, &a.Type, &a.TakenAt, &a.Width, &a.Height, &a.Duration, &a.Path, &a.Size); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// GET /api/timeline?limit=&before=&before_id=  (terbaru dulu, keyset pagination)
func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 200)
	if limit < 1 || limit > 500 {
		limit = 200
	}
	// order=asc: terlama dulu. Kursor (before/before_id) berarti "setelah" pada urutan itu.
	asc := r.URL.Query().Get("order") == "asc"
	startAt, startID := int64(math.MaxInt64), int64(math.MaxInt64)
	cmp, dir := "<", "DESC"
	if asc {
		startAt, startID = math.MinInt64, math.MinInt64
		cmp, dir = ">", "ASC"
	}
	before := queryInt(r, "before", startAt)
	beforeID := queryInt(r, "before_id", startID)
	// Rentang waktu opsional (detik unix): from inklusif, to eksklusif.
	from := queryInt(r, "from", math.MinInt64)
	to := queryInt(r, "to", math.MaxInt64)

	// ids=1,2,3 membatasi hasil ke id tertentu (dipakai halaman Favorit).
	idFilter := ""
	args := []any{from, to, before, before, beforeID}
	if raw, ok := r.URL.Query()["ids"]; ok {
		var ph []string
		for _, part := range strings.Split(strings.Join(raw, ","), ",") {
			if n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && len(ph) < 2000 {
				ph = append(ph, "?")
				args = append(args, n)
			}
		}
		if len(ph) == 0 {
			writeJSON(w, 200, map[string]any{"items": []assetItem{}, "next": nil})
			return
		}
		idFilter = " AND id IN (" + strings.Join(ph, ",") + ")"
	}
	args = append(args, limit)

	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT id, type, taken_at, width, height, duration, path, size FROM asset
		WHERE deleted_at IS NULL AND taken_at >= ? AND taken_at < ?
		  AND (taken_at `+cmp+` ? OR (taken_at = ? AND id `+cmp+` ?))`+idFilter+`
		ORDER BY taken_at `+dir+`, id `+dir+` LIMIT ?`, args...)
	if err != nil {
		serverErr(w, err)
		return
	}
	items, err := scanItems(rows)
	if err != nil {
		serverErr(w, err)
		return
	}
	var next *cursor
	if int64(len(items)) == limit {
		last := items[len(items)-1]
		next = &cursor{Before: last.TakenAt, BeforeID: last.ID}
	}
	writeJSON(w, 200, map[string]any{"items": items, "next": next})
}

// asset mengembalikan metadata satu aset untuk deep-link /photos/:id.
func (s *Server) asset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpErr(w, http.StatusNotFound, "not found")
		return
	}
	var a assetItem
	err = s.DB.QueryRowContext(r.Context(), `
		SELECT id, type, taken_at, width, height, duration, path, size FROM asset
		WHERE id = ? AND deleted_at IS NULL`, id).
		Scan(&a.ID, &a.Type, &a.TakenAt, &a.Width, &a.Height, &a.Duration, &a.Path, &a.Size)
	if errors.Is(err, sql.ErrNoRows) {
		httpErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}
