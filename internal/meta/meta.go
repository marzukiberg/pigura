// Package meta membaca metadata foto/video (EXIF, GPS, codec) lewat vipsheader dan ffprobe.
// Parser murni (tanpa exec) sehingga mudah diuji; pembacaan file dibungkus Reader.
package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Info metadata satu aset. Field kosong/nol berarti tidak tersedia.
type Info struct {
	Kind        string    `json:"kind"` // image | video
	Format      string    `json:"format,omitempty"`
	Width       int       `json:"width,omitempty"` // piksel sebenarnya, sudah memperhitungkan orientasi
	Height      int       `json:"height,omitempty"`
	Megapixels  float64   `json:"megapixels,omitempty"`
	Orientation int       `json:"orientation,omitempty"`
	TakenLocal  string    `json:"taken_local,omitempty"` // waktu ambil dari EXIF, tanpa zona waktu ("2026-02-22 18:33:49")
	TakenOffset string    `json:"taken_offset,omitempty"`
	Camera      *Camera   `json:"camera,omitempty"`
	Exposure    *Exposure `json:"exposure,omitempty"`
	GPS         *GPS      `json:"gps,omitempty"`
	Video       *Video    `json:"video,omitempty"`
}

type Camera struct {
	Make     string `json:"make,omitempty"`
	Model    string `json:"model,omitempty"`
	Lens     string `json:"lens,omitempty"`
	Software string `json:"software,omitempty"`
}

type Exposure struct {
	ISO         int     `json:"iso,omitempty"`
	FNumber     float64 `json:"f_number,omitempty"`
	Shutter     string  `json:"shutter,omitempty"` // "1/33"
	FocalLength float64 `json:"focal_length,omitempty"`
	Focal35     int     `json:"focal_35,omitempty"`
	Flash       string  `json:"flash,omitempty"`
	Bias        string  `json:"bias,omitempty"`
	Metering    string  `json:"metering,omitempty"`
	Program     string  `json:"program,omitempty"`
	WhiteBal    string  `json:"white_balance,omitempty"`
}

type GPS struct {
	Lat float64  `json:"lat"`
	Lon float64  `json:"lon"`
	Alt *float64 `json:"alt,omitempty"`
}

type Video struct {
	Duration   float64 `json:"duration,omitempty"`
	Codec      string  `json:"codec,omitempty"`
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	FPS        float64 `json:"fps,omitempty"`
	Bitrate    int64   `json:"bitrate,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	Rotation   int     `json:"rotation,omitempty"`
	Created    string  `json:"created,omitempty"` // creation_time dari container
}

// ---------- parser gambar (keluaran `vipsheader -a`) ----------

// exif-ifd2-FNumber: 1800/1000 (f/1.8, Rational, 1 components, 8 bytes)
var exifLine = regexp.MustCompile(`^exif-ifd\d-(\w+): (.*?) \((.*), \w+, \d+ components, \d+ bytes\)$`)

type exifField struct{ raw, pretty string }

func exifFields(out string) (map[string]exifField, map[string]string) {
	ex := map[string]exifField{}
	plain := map[string]string{}
	for _, ln := range strings.Split(out, "\n") {
		if m := exifLine.FindStringSubmatch(ln); m != nil {
			ex[m[1]] = exifField{strings.TrimSpace(m[2]), strings.TrimSpace(m[3])}
		} else if k, v, ok := strings.Cut(ln, ": "); ok && !strings.HasPrefix(k, "exif-") {
			plain[k] = strings.TrimSpace(v)
		}
	}
	return ex, plain
}

// rational mengurai "a/b" atau bilangan biasa.
func rational(s string) (float64, bool) {
	if n, d, ok := strings.Cut(strings.TrimSpace(s), "/"); ok {
		a, e1 := strconv.ParseFloat(n, 64)
		b, e2 := strconv.ParseFloat(d, 64)
		if e1 != nil || e2 != nil || b == 0 {
			return 0, false
		}
		return a / b, true
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f, err == nil
}

// dms mengubah "d/1 m/1 s/10000" + referensi (N/S/E/W) menjadi derajat desimal.
func dms(raw, ref string) (float64, bool) {
	parts := strings.Fields(raw)
	if len(parts) != 3 {
		return 0, false
	}
	var v [3]float64
	for i, p := range parts {
		x, ok := rational(p)
		if !ok {
			return 0, false
		}
		v[i] = x
	}
	deg := v[0] + v[1]/60 + v[2]/3600
	if ref == "S" || ref == "W" {
		deg = -deg
	}
	return deg, true
}

func cleanStr(s string) string { return strings.TrimSpace(strings.Trim(s, "\x00")) }

var shutterRe = regexp.MustCompile(`^(\d+)/(\d+) sec`)

// ParseImage mengurai keluaran `vipsheader -a`.
func ParseImage(out string) *Info {
	ex, plain := exifFields(out)
	info := &Info{Kind: "image", Format: strings.TrimSuffix(plain["vips-loader"], "load")}
	if c := plain["heif-compression"]; c != "" {
		info.Format = "heif/" + c
	}
	w, _ := strconv.Atoi(plain["width"])
	h, _ := strconv.Atoi(plain["height"])
	if o, ok := ex["Orientation"]; ok {
		info.Orientation, _ = strconv.Atoi(o.raw)
	}
	if info.Orientation >= 5 && info.Orientation <= 8 { // diputar 90°: tukar sisi
		w, h = h, w
	}
	info.Width, info.Height = w, h
	if w > 0 && h > 0 {
		info.Megapixels = math.Round(float64(w)*float64(h)/1e5) / 10
	}

	if v, ok := ex["DateTimeOriginal"]; ok {
		if t, err := time.Parse("2006:01:02 15:04:05", v.pretty); err == nil {
			info.TakenLocal = t.Format("2006-01-02 15:04:05")
		}
	}
	info.TakenOffset = ex["OffsetTimeOriginal"].pretty

	cam := &Camera{Make: cleanStr(ex["Make"].pretty), Model: cleanStr(ex["Model"].pretty),
		Lens: cleanStr(ex["LensModel"].pretty), Software: cleanStr(ex["Software"].pretty)}
	if cam.Model != "" && cam.Make != "" && strings.HasPrefix(strings.ToLower(cam.Model), strings.ToLower(cam.Make)) {
		cam.Make = "" // model sudah memuat merek ("Apple iPhone 15" vs make "Apple")
	}
	if *cam != (Camera{}) {
		info.Camera = cam
	}

	e := &Exposure{}
	if v, ok := ex["ISOSpeedRatings"]; ok {
		e.ISO, _ = strconv.Atoi(v.raw)
	} else if v, ok := ex["PhotographicSensitivity"]; ok {
		e.ISO, _ = strconv.Atoi(v.raw)
	}
	if v, ok := ex["FNumber"]; ok {
		e.FNumber, _ = rational(v.raw)
		e.FNumber = math.Round(e.FNumber*10) / 10
	}
	if v, ok := ex["ExposureTime"]; ok {
		if m := shutterRe.FindStringSubmatch(v.pretty); m != nil {
			e.Shutter = m[1] + "/" + m[2]
		} else if f, ok := rational(v.raw); ok && f > 0 {
			e.Shutter = strconv.FormatFloat(math.Round(f*10)/10, 'f', -1, 64)
		}
	}
	if v, ok := ex["FocalLength"]; ok {
		e.FocalLength, _ = rational(v.raw)
		e.FocalLength = math.Round(e.FocalLength*10) / 10
	}
	if v, ok := ex["FocalLengthIn35mmFilm"]; ok {
		e.Focal35, _ = strconv.Atoi(v.raw)
	}
	if v, ok := ex["Flash"]; ok {
		e.Flash = tr.flash(v.pretty)
	}
	if v, ok := ex["ExposureBiasValue"]; ok {
		e.Bias = v.pretty
	}
	e.Metering = tr.translate(ex["MeteringMode"].pretty)
	e.Program = tr.translate(ex["ExposureProgram"].pretty)
	e.WhiteBal = tr.translate(ex["WhiteBalance"].pretty)
	if *e != (Exposure{}) {
		info.Exposure = e
	}

	if lat, ok := dms(ex["GPSLatitude"].raw, ex["GPSLatitudeRef"].pretty); ok {
		if lon, ok := dms(ex["GPSLongitude"].raw, ex["GPSLongitudeRef"].pretty); ok && (lat != 0 || lon != 0) {
			g := &GPS{Lat: round6(lat), Lon: round6(lon)}
			if a, ok := rational(ex["GPSAltitude"].raw); ok {
				if strings.Contains(strings.ToLower(ex["GPSAltitudeRef"].pretty), "below") {
					a = -a
				}
				a = math.Round(a*10) / 10
				g.Alt = &a
			}
			info.GPS = g
		}
	}
	return info
}

func round6(f float64) float64 { return math.Round(f*1e6) / 1e6 }

// terjemahan nilai EXIF umum ke bahasa Indonesia
type translator struct{}

var tr translator

var idMap = map[string]string{
	"Auto white balance": "Otomatis", "Manual white balance": "Manual",
	"Normal program": "Program normal", "Manual": "Manual", "Aperture priority": "Prioritas bukaan",
	"Shutter priority": "Prioritas shutter", "Creative program": "Program kreatif", "Action program": "Program aksi",
	"Portrait mode": "Mode potret", "Landscape mode": "Mode lanskap",
	"Center-weighted average": "Rata-rata terpusat", "Multi-segment": "Multi-segmen", "Pattern": "Pola",
	"Spot": "Titik", "Average": "Rata-rata", "Partial": "Sebagian", "Multi-spot": "Multi-titik",
}

func (translator) translate(s string) string {
	if s == "" {
		return ""
	}
	if t, ok := idMap[s]; ok {
		return t
	}
	return s
}

func (translator) flash(s string) string {
	l := strings.ToLower(s)
	switch {
	case s == "":
		return ""
	case strings.Contains(l, "did not fire"), strings.Contains(l, "no flash"):
		return "Tidak menyala"
	case strings.Contains(l, "fired"), strings.Contains(l, "flash"):
		return "Menyala"
	}
	return s
}

// ---------- parser video (keluaran `ffprobe -print_format json`) ----------

type probe struct {
	Format struct {
		Duration string            `json:"duration"`
		BitRate  string            `json:"bit_rate"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		Type       string            `json:"codec_type"`
		Codec      string            `json:"codec_name"`
		Width      int               `json:"width"`
		Height     int               `json:"height"`
		RFrameRate string            `json:"r_frame_rate"`
		Tags       map[string]string `json:"tags"`
		SideData   []struct {
			Rotation float64 `json:"rotation"`
		} `json:"side_data_list"`
	} `json:"streams"`
}

// ISO 6709: "+28.4052+101.3987+035.000/"
var iso6709 = regexp.MustCompile(`^([+-]\d+(?:\.\d+)?)([+-]\d+(?:\.\d+)?)`)

// ParseVideo mengurai keluaran ffprobe JSON.
func ParseVideo(data []byte) (*Info, error) {
	var p probe
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	info := &Info{Kind: "video"}
	v := &Video{Created: p.Format.Tags["creation_time"]}
	v.Duration, _ = strconv.ParseFloat(p.Format.Duration, 64)
	v.Duration = math.Round(v.Duration*10) / 10
	v.Bitrate, _ = strconv.ParseInt(p.Format.BitRate, 10, 64)
	for _, s := range p.Streams {
		switch s.Type {
		case "video":
			if v.Codec != "" {
				continue
			}
			v.Codec, v.Width, v.Height = s.Codec, s.Width, s.Height
			if f, ok := rational(s.RFrameRate); ok {
				v.FPS = math.Round(f*100) / 100
			}
			for _, sd := range s.SideData {
				if sd.Rotation != 0 {
					v.Rotation = ((int(sd.Rotation) % 360) + 360) % 360
				}
			}
			if r := s.Tags["rotate"]; r != "" && v.Rotation == 0 {
				v.Rotation, _ = strconv.Atoi(r)
			}
		case "audio":
			if v.AudioCodec == "" {
				v.AudioCodec = s.Codec
			}
		}
	}
	w, h := v.Width, v.Height
	if v.Rotation == 90 || v.Rotation == 270 {
		w, h = h, w
	}
	info.Width, info.Height = w, h
	if w > 0 && h > 0 {
		info.Megapixels = math.Round(float64(w)*float64(h)/1e5) / 10
	}
	loc := p.Format.Tags["location"]
	if loc == "" {
		loc = p.Format.Tags["com.apple.quicktime.location.ISO6709"]
	}
	if m := iso6709.FindStringSubmatch(loc); m != nil {
		lat, _ := strconv.ParseFloat(m[1], 64)
		lon, _ := strconv.ParseFloat(m[2], 64)
		if lat != 0 || lon != 0 {
			info.GPS = &GPS{Lat: round6(lat), Lon: round6(lon)}
		}
	}
	info.Video = v
	return info, nil
}

// ---------- pembaca (exec) ----------

// Runner menjalankan program dan mengembalikan stdout; diganti di test.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// Reader membaca metadata satu file.
type Reader struct {
	Run     Runner
	Timeout time.Duration
}

func NewReader() *Reader { return &Reader{Run: execRunner, Timeout: 12 * time.Second} }

// DefaultReader memakai vipsheader/ffprobe dari PATH.
var DefaultReader = NewReader()

// Read membaca metadata. path harus absolut (bukan opsi bagi program).
func (r *Reader) Read(ctx context.Context, path, kind string) (*Info, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	if kind == "video" {
		out, err := r.Run(ctx, "ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
		if err != nil {
			return nil, fmt.Errorf("ffprobe: %w", err)
		}
		return ParseVideo(out)
	}
	out, err := r.Run(ctx, "vipsheader", "-a", path)
	if err != nil {
		return nil, fmt.Errorf("vipsheader: %w", err)
	}
	return ParseImage(string(out)), nil
}
