// Album pengguna.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (s *Server) albums(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT a.id, a.name, a.created_at,
		       COALESCE(a.cover_id, (SELECT asset_id FROM album_asset
		                             WHERE album_id = a.id ORDER BY position, asset_id LIMIT 1)),
		       (SELECT COUNT(*) FROM album_asset x WHERE x.album_id = a.id)
		FROM album a ORDER BY a.created_at DESC, a.id DESC`)
	if err != nil {
		serverErr(w, err)
		return
	}
	defer rows.Close()
	type album struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		CreatedAt int64  `json:"created_at"`
		CoverID   *int64 `json:"cover_id"`
		Count     int    `json:"count"`
	}
	out := []album{}
	for rows.Next() {
		var a album
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt, &a.CoverID, &a.Count); err != nil {
			serverErr(w, err)
			return
		}
		out = append(out, a)
	}
	writeJSON(w, 200, out)
}

func (s *Server) createAlbum(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil ||
		strings.TrimSpace(in.Name) == "" {
		httpErr(w, 400, "name required")
		return
	}
	res, err := s.DB.ExecContext(r.Context(),
		`INSERT INTO album (name, created_at) VALUES (?, ?)`, strings.TrimSpace(in.Name), time.Now().Unix())
	if err != nil {
		serverErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, 201, map[string]any{"id": id})
}

func (s *Server) albumAssets(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT a.id, a.type, a.taken_at, a.width, a.height, a.duration, a.path, a.size
		FROM album_asset x JOIN asset a ON a.id = x.asset_id
		WHERE x.album_id = ? AND a.deleted_at IS NULL
		ORDER BY a.taken_at DESC, a.id DESC`, id)
	if err != nil {
		serverErr(w, err)
		return
	}
	items, err := scanItems(rows)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) addAlbumAssets(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil || len(in.IDs) == 0 {
		httpErr(w, 400, "ids required")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		serverErr(w, err)
		return
	}
	defer tx.Rollback()
	added := 0
	for _, aid := range in.IDs {
		res, err := tx.ExecContext(r.Context(), `
			INSERT OR IGNORE INTO album_asset (album_id, asset_id, position)
			SELECT ?, id, ? FROM asset WHERE id = ?`, id, added, aid)
		if err != nil {
			serverErr(w, err)
			return
		}
		if n, _ := res.RowsAffected(); n > 0 {
			added++
		}
	}
	if err := tx.Commit(); err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, 200, map[string]int{"added": added})
}
