// Mutasi aset (hapus lunak).
package server

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) deleteAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	res, err := s.DB.ExecContext(r.Context(),
		`UPDATE asset SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`, time.Now().Unix(), id)
	if err != nil {
		serverErr(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": n})
}

type batchDeleteReq struct {
	IDs []int64 `json:"ids"`
}

func (s *Server) batchDeleteAssets(w http.ResponseWriter, r *http.Request) {
	var req batchDeleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		httpErr(w, http.StatusBadRequest, "invalid or empty ids")
		return
	}
	if len(req.IDs) > 1000 {
		httpErr(w, http.StatusBadRequest, "too many ids (max 1000)")
		return
	}
	now := time.Now().Unix()
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		serverErr(w, err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(r.Context(), `UPDATE asset SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`)
	if err != nil {
		serverErr(w, err)
		return
	}
	defer stmt.Close()

	var count int64
	for _, id := range req.IDs {
		res, err := stmt.ExecContext(r.Context(), now, id)
		if err == nil {
			if affected, _ := res.RowsAffected(); affected > 0 {
				count += affected
			}
		}
	}
	if err := tx.Commit(); err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": count})
}
