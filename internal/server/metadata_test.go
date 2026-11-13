package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/marzukiberg/pigura/internal/meta"
)

func TestAssetMetadataEndpoint(t *testing.T) {
	s, _, pub, ids := shareFixture(t)
	var calls atomic.Int32
	var lastName, lastPath string
	s.Meta = &meta.Reader{Timeout: 2e9, Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls.Add(1)
		lastName, lastPath = name, args[len(args)-1]
		if name == "vipsheader" {
			return []byte("width: 40\nheight: 30\nvips-loader: jpegload\nexif-ifd0-Make: Canon (Canon, ASCII, 6 components, 6 bytes)\nexif-ifd0-Model: Canon EOS R5 (Canon EOS R5, ASCII, 12 components, 12 bytes)\n"), nil
		}
		return []byte(`{"format":{"duration":"3.2"},"streams":[{"codec_type":"video","codec_name":"h264","width":640,"height":360,"r_frame_rate":"30/1"}]}`), nil
	}}
	h := s.Handler()
	get := func(id int64) (*httpRecorder, map[string]any) {
		rec := do(h, "GET", "/api/assets/"+itoa(id)+"/metadata", "localhost", nil, "")
		var m map[string]any
		json.Unmarshal(rec.Body.Bytes(), &m)
		return &httpRecorder{rec.Code}, m
	}

	rec, m := get(ids["a.jpg"])
	if rec.code != 200 || m["kind"] != "image" {
		t.Fatalf("image = %d %v", rec.code, m)
	}
	cam := m["camera"].(map[string]any)
	if cam["model"] != "Canon EOS R5" || cam["make"] != nil { // merek dibuang: model sudah memuatnya
		t.Errorf("camera = %v", cam)
	}
	if lastName != "vipsheader" || !strings.HasSuffix(lastPath, "/a.jpg") || !strings.HasPrefix(lastPath, "/") {
		t.Errorf("perintah = %s %s (harus path absolut milik library)", lastName, lastPath)
	}

	// panggilan kedua dilayani dari cache (tanpa menjalankan tool lagi)
	before := calls.Load()
	get(ids["a.jpg"])
	if calls.Load() != before {
		t.Error("hasil seharusnya di-cache per (id, mtime)")
	}

	// video memakai ffprobe
	_, mv := get(ids["b.mp4"])
	if lastName != "ffprobe" || mv["kind"] != "video" || mv["video"].(map[string]any)["codec"] != "h264" {
		t.Errorf("video = %s %v", lastName, mv)
	}

	// tool gagal: tetap 200 dengan info minimal (panel tidak error), dan tidak di-cache
	s.Meta.Run = func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("vipsheader: tidak ada")
	}
	s.metaCache = metaCache{}
	rec, m = get(ids["a.jpg"])
	if rec.code != 200 || m["kind"] != "image" || m["camera"] != nil {
		t.Errorf("galat tool = %d %v", rec.code, m)
	}

	// id tak ada / bukan angka
	if c := do(h, "GET", "/api/assets/999999/metadata", "localhost", nil, "").Code; c != 404 {
		t.Errorf("id tak ada = %d, want 404", c)
	}

	// METADATA HANYA UNTUK PEMILIK: tidak tersedia di listener publik
	if c := do(pub, "GET", "/api/assets/"+itoa(ids["a.jpg"])+"/metadata", "foto.contoh.id", nil, "").Code; c != http.StatusNotFound {
		t.Errorf("listener publik = %d, want 404 (EXIF/GPS bersifat privat)", c)
	}
	// ...dan butuh token pemilik bila diaktifkan
	s.Token = "t0ken"
	if c := do(h, "GET", "/api/assets/"+itoa(ids["a.jpg"])+"/metadata", "localhost", nil, "").Code; c != 401 {
		t.Errorf("tanpa token = %d, want 401", c)
	}
}

type httpRecorder struct{ code int }
