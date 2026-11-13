// Model tautan berbagi: kunci acak, password (PBKDF2), status.
package server

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	pwIter          = 300_000
	maxShareTTL     = 365 * 24 * 3600
	unlockCookieTTL = 12 * 3600
	maxShareItems   = 2000
	keyLen          = 22 // 16 byte -> base64url tanpa padding
)

func newShareKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func validShareKey(k string) bool {
	if len(k) != keyLen {
		return false
	}
	for _, c := range k {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func hashSharePassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk, err := pbkdf2.Key(sha256.New, pw, salt, pwIter, 32)
	if err != nil {
		return "", err
	}
	e := base64.RawStdEncoding
	return fmt.Sprintf("p1$%d$%s$%s", pwIter, e.EncodeToString(salt), e.EncodeToString(dk)), nil
}

func checkSharePassword(enc, pw string) bool {
	parts := strings.Split(enc, "$")
	if len(parts) != 4 || parts[0] != "p1" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1000 || iter > 5_000_000 {
		return false
	}
	e := base64.RawStdEncoding
	salt, err1 := e.DecodeString(parts[2])
	want, err2 := e.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

type shareRow struct {
	ID            int64
	Key, Kind     string
	TargetID      int64
	Title         string
	CreatedAt     int64
	ExpiresAt     sql.NullInt64
	PW            sql.NullString
	AllowDownload bool
	Views         int64
	LastViewedAt  sql.NullInt64
	RevokedAt     sql.NullInt64
}

const shareCols = `id, key, kind, target_id, title, created_at, expires_at, pw, allow_download, views, last_viewed_at, revoked_at`

func scanShare(sc interface{ Scan(...any) error }) (*shareRow, error) {
	var r shareRow
	var dl int
	if err := sc.Scan(&r.ID, &r.Key, &r.Kind, &r.TargetID, &r.Title, &r.CreatedAt, &r.ExpiresAt, &r.PW,
		&dl, &r.Views, &r.LastViewedAt, &r.RevokedAt); err != nil {
		return nil, err
	}
	r.AllowDownload = dl == 1
	return &r, nil
}

func (r *shareRow) status(now int64) string {
	switch {
	case r.RevokedAt.Valid:
		return "revoked"
	case r.ExpiresAt.Valid && r.ExpiresAt.Int64 <= now:
		return "expired"
	}
	return "active"
}
