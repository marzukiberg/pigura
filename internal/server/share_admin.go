// API pemilik untuk tautan berbagi (butuh token pemilik bila aktif).
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"time"

	"github.com/marzukiberg/pigura/internal/thumbs"
)

func (s *Server) targetInfo(kind string, id int64) (title string, ok bool) {
	switch kind {
	case "album":
		err := s.DB.QueryRow(`SELECT name FROM album WHERE id = ?`, id).Scan(&title)
		return title, err == nil
	case "asset":
		var p string
		err := s.DB.QueryRow(`SELECT path FROM asset WHERE id = ? AND deleted_at IS NULL`, id).Scan(&p)
		return path.Base(p), err == nil
	}
	return "", false
}

// POST /api/shares
func (s *Server) createShare(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind          string `json:"kind"`
		TargetID      int64  `json:"target_id"`
		Password      string `json:"password"`
		ExpiresIn     int64  `json:"expires_in"` // detik; 0 = tidak kedaluwarsa
		AllowDownload bool   `json:"allow_download"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if in.ExpiresIn < 0 || in.ExpiresIn > maxShareTTL {
		httpErr(w, http.StatusBadRequest, "expires_in must be between 0 and 365 days")
		return
	}
	if in.Password != "" && (len(in.Password) < 4 || len(in.Password) > 200) {
		httpErr(w, http.StatusBadRequest, "password must be 4-200 characters")
		return
	}
	title, ok := s.targetInfo(in.Kind, in.TargetID)
	if !ok {
		httpErr(w, http.StatusNotFound, "target not found")
		return
	}
	key, err := newShareKey()
	if err != nil {
		serverErr(w, err)
		return
	}
	var pw any
	if in.Password != "" {
		enc, err := hashSharePassword(in.Password)
		if err != nil {
			serverErr(w, err)
			return
		}
		pw = enc
	}
	now := time.Now().Unix()
	var exp any
	var expAt int64
	if in.ExpiresIn > 0 {
		expAt = now + in.ExpiresIn
		exp = expAt
	}
	dl := 0
	if in.AllowDownload {
		dl = 1
	}
	res, err := s.DB.Exec(`INSERT INTO share (key, kind, target_id, title, created_at, expires_at, pw, allow_download)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, key, in.Kind, in.TargetID, title, now, exp, pw, dl)
	if err != nil {
		serverErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	s.prewarmShare(in.Kind, in.TargetID)
	urlPath := "/s/" + key
	out := map[string]any{"id": id, "key": key, "path": urlPath, "public": s.PublicURL != "",
		"url": s.PublicURL + urlPath, "has_password": in.Password != "", "allow_download": in.AllowDownload}
	if expAt > 0 {
		out["expires_at"] = expAt
	}
	writeJSON(w, http.StatusCreated, out)
}

// prewarmShare membuat thumbnail aset yang dibagikan di latar belakang (prioritas rendah)
// agar halaman publik tidak memicu konversi berat saat dikunjungi banyak orang.
func (s *Server) prewarmShare(kind string, id int64) {
	if s.Gen == nil || s.Library == "" {
		return
	}
	var q string
	switch kind {
	case "album":
		q = `SELECT a.id, a.path, a.type, a.mtime FROM album_asset x JOIN asset a ON a.id = x.asset_id
			WHERE x.album_id = ? AND a.deleted_at IS NULL LIMIT 500`
	default:
		q = `SELECT id, path, type, mtime FROM asset WHERE id = ? AND deleted_at IS NULL`
	}
	rows, err := s.DB.Query(q, id)
	if err != nil {
		return
	}
	var list []assetRow
	for rows.Next() {
		var a assetRow
		if rows.Scan(&a.ID, &a.Rel, &a.Type, &a.Mtime) == nil {
			list = append(list, a)
		}
	}
	rows.Close()
	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		ctx := thumbs.Background(context.Background())
		for _, a := range list {
			src, err := safeJoin(s.Library, a.Rel)
			if err != nil || !s.insideLibrary(src) {
				continue
			}
			s.Gen.Ensure(ctx, "sthumb", a.ID, src, a.Type == "video", a.Mtime, thumbs.ThumbEdge)
		}
	}()
}

// GET /api/shares
func (s *Server) listShares(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT `+shareCols+` FROM share ORDER BY created_at DESC, id DESC LIMIT 500`)
	if err != nil {
		serverErr(w, err)
		return
	}
	defer rows.Close()
	now := time.Now().Unix()
	out := []map[string]any{}
	for rows.Next() {
		sh, err := scanShare(rows)
		if err != nil {
			serverErr(w, err)
			return
		}
		item := map[string]any{
			"id": sh.ID, "key": sh.Key, "kind": sh.Kind, "target_id": sh.TargetID, "title": sh.Title,
			"created_at": sh.CreatedAt, "has_password": sh.PW.Valid, "allow_download": sh.AllowDownload,
			"views": sh.Views, "status": sh.status(now), "path": "/s/" + sh.Key, "url": s.PublicURL + "/s/" + sh.Key,
		}
		if sh.ExpiresAt.Valid {
			item["expires_at"] = sh.ExpiresAt.Int64
		}
		if sh.LastViewedAt.Valid {
			item["last_viewed_at"] = sh.LastViewedAt.Int64
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"public": s.PublicURL != "", "items": out})
}

// DELETE /api/shares/{id} — cabut tautan (langsung berhenti berlaku).
func (s *Server) revokeShare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE share SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, time.Now().Unix(), id)
	if err != nil {
		serverErr(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
