package meta

import (
	"context"
	"errors"
	"testing"
)

// Keluaran nyata `vipsheader -a` dari foto Motorola (disingkat).
const sampleJPEG = `/x/IMG.jpg: 8960x6144 uchar, 3 bands, srgb, jpegload
width: 8960
height: 6144
vips-loader: jpegload
exif-ifd0-Make: motorola (motorola, ASCII, 32 components, 32 bytes)
exif-ifd0-Orientation: 6 (Right-top, Short, 1 components, 2 bytes)
exif-ifd0-Model: motorola edge 60 fusion (motorola edge 60 fusion, ASCII, 32 components, 32 bytes)
exif-ifd0-Software:  (, ASCII, 32 components, 32 bytes)
exif-ifd2-ExposureBiasValue: 0/100 (0.00 EV, SRational, 1 components, 8 bytes)
exif-ifd2-ExposureProgram: 2 (Normal program, Short, 1 components, 2 bytes)
exif-ifd2-DateTimeOriginal: 2026:02:22 18:33:49 (2026:02:22 18:33:49, ASCII, 20 components, 20 bytes)
exif-ifd2-WhiteBalance: 0 (Auto white balance, Short, 1 components, 2 bytes)
exif-ifd2-ExposureTime: 30001856/1000000000 (1/33 sec., Rational, 1 components, 8 bytes)
exif-ifd2-Flash: 0 (Flash did not fire, Short, 1 components, 2 bytes)
exif-ifd2-FNumber: 1800/1000 (f/1.8, Rational, 1 components, 8 bytes)
exif-ifd2-ISOSpeedRatings: 1969 (1969, Short, 1 components, 2 bytes)
exif-ifd2-FocalLengthIn35mmFilm: 24 (24, Short, 1 components, 2 bytes)
exif-ifd2-MeteringMode: 2 (Center-weighted average, Short, 1 components, 2 bytes)
exif-ifd2-FocalLength: 5560/1000 (5.6 mm, Rational, 1 components, 8 bytes)
exif-ifd3-GPSLatitude: 0/1 28/1 187139/10000 ( 0, 28, 18.7139, Rational, 3 components, 24 bytes)
exif-ifd3-GPSAltitude: 35/1 (35, Rational, 1 components, 8 bytes)
exif-ifd3-GPSLatitudeRef: N (N, ASCII, 2 components, 2 bytes)
exif-ifd3-GPSAltitudeRef: Sea level (Sea level, Byte, 1 components, 1 bytes)
exif-ifd3-GPSLongitudeRef: E (E, ASCII, 2 components, 2 bytes)
exif-ifd3-GPSLongitude: 101/1 24/1 52789/10000 (101, 24, 5.2789, Rational, 3 components, 24 bytes)
`

func TestParseImage(t *testing.T) {
	i := ParseImage(sampleJPEG)
	if i.Kind != "image" || i.Format != "jpeg" {
		t.Fatalf("kind/format = %s/%s", i.Kind, i.Format)
	}
	// orientasi 6 = diputar 90°: 8960x6144 menjadi 6144x8960
	if i.Width != 6144 || i.Height != 8960 || i.Orientation != 6 || i.Megapixels != 55.1 {
		t.Errorf("dimensi = %dx%d o=%d mp=%v", i.Width, i.Height, i.Orientation, i.Megapixels)
	}
	if i.TakenLocal != "2026-02-22 18:33:49" {
		t.Errorf("taken = %q", i.TakenLocal)
	}
	if i.Camera == nil || i.Camera.Model != "motorola edge 60 fusion" || i.Camera.Make != "" || i.Camera.Software != "" {
		t.Errorf("camera = %+v (make dibuang karena model sudah memuatnya; software kosong)", i.Camera)
	}
	e := i.Exposure
	if e == nil || e.ISO != 1969 || e.FNumber != 1.8 || e.Shutter != "1/33" || e.FocalLength != 5.6 || e.Focal35 != 24 {
		t.Fatalf("exposure = %+v", e)
	}
	if e.Flash != "Tidak menyala" || e.Metering != "Rata-rata terpusat" || e.Program != "Program normal" || e.WhiteBal != "Otomatis" || e.Bias != "0.00 EV" {
		t.Errorf("exposure teks = %+v", e)
	}
	g := i.GPS
	if g == nil || g.Lat != 0.471865 || g.Lon != 101.401466 || g.Alt == nil || *g.Alt != 35 {
		t.Fatalf("gps = %+v (lat 0°28'18.7139\" N, lon 101°24'5.2789\" E)", g)
	}
}

func TestParseImageHemispheresAndEmpty(t *testing.T) {
	s := "exif-ifd3-GPSLatitude: 6/1 10/1 0/1 (6, 10, 0, Rational, 3 components, 24 bytes)\n" +
		"exif-ifd3-GPSLatitudeRef: S (S, ASCII, 2 components, 2 bytes)\n" +
		"exif-ifd3-GPSLongitude: 106/1 49/1 30/1 (106, 49, 30, Rational, 3 components, 24 bytes)\n" +
		"exif-ifd3-GPSLongitudeRef: W (W, ASCII, 2 components, 2 bytes)\n" +
		"exif-ifd3-GPSAltitude: 5/1 (5, Rational, 1 components, 8 bytes)\n" +
		"exif-ifd3-GPSAltitudeRef: Below sea level (Below sea level, Byte, 1 components, 1 bytes)\n"
	g := ParseImage(s).GPS
	if g == nil || g.Lat >= 0 || g.Lon >= 0 || g.Alt == nil || *g.Alt != -5 {
		t.Fatalf("belahan bumi/ketinggian salah: %+v", g)
	}
	// tanpa EXIF sama sekali: tidak boleh panik, bagian opsional nil
	e := ParseImage("width: 10\nheight: 20\nvips-loader: pngload\n")
	if e.Camera != nil || e.Exposure != nil || e.GPS != nil || e.Width != 10 || e.Format != "png" {
		t.Fatalf("tanpa exif = %+v", e)
	}
	// GPS 0,0 (nilai kosong umum) tidak dianggap lokasi
	z := ParseImage("exif-ifd3-GPSLatitude: 0/1 0/1 0/1 (0, 0, 0, Rational, 3 components, 24 bytes)\nexif-ifd3-GPSLongitude: 0/1 0/1 0/1 (0, 0, 0, Rational, 3 components, 24 bytes)\n")
	if z.GPS != nil {
		t.Errorf("gps 0,0 harus diabaikan: %+v", z.GPS)
	}
}

func TestParseVideo(t *testing.T) {
	j := `{"format":{"duration":"48.565000","bit_rate":"15227234","tags":{"creation_time":"2025-04-25T13:16:08.000000Z","location":"+28.4052+101.3987/"}},
	"streams":[{"codec_type":"video","codec_name":"hevc","width":888,"height":1920,"r_frame_rate":"60000/1001","side_data_list":[{"rotation":-90}]},
	{"codec_type":"audio","codec_name":"aac"}]}`
	i, err := ParseVideo([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	v := i.Video
	if v == nil || v.Codec != "hevc" || v.AudioCodec != "aac" || v.Duration != 48.6 || v.Bitrate != 15227234 || v.FPS != 59.94 || v.Rotation != 270 {
		t.Fatalf("video = %+v", v)
	}
	if i.Width != 1920 || i.Height != 888 { // rotasi 270 menukar sisi
		t.Errorf("dimensi = %dx%d", i.Width, i.Height)
	}
	if i.GPS == nil || i.GPS.Lat != 28.4052 || i.GPS.Lon != 101.3987 {
		t.Errorf("gps video = %+v", i.GPS)
	}
	if _, err := ParseVideo([]byte("bukan json")); err == nil {
		t.Error("json rusak harus galat")
	}
}

func TestReaderUsesRunnerAndPropagatesErrors(t *testing.T) {
	var gotName string
	var gotArgs []string
	r := &Reader{Timeout: 2e9, Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		gotName, gotArgs = name, args
		return []byte(sampleJPEG), nil
	}}
	if i, err := r.Read(context.Background(), "/mnt/x.jpg", "image"); err != nil || i.Camera == nil {
		t.Fatalf("read = %v %v", i, err)
	}
	if gotName != "vipsheader" || gotArgs[len(gotArgs)-1] != "/mnt/x.jpg" {
		t.Errorf("perintah = %s %v", gotName, gotArgs)
	}
	r.Run = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("tidak ada") }
	if _, err := r.Read(context.Background(), "/mnt/v.mp4", "video"); err == nil {
		t.Error("galat tool harus diteruskan")
	}
}
