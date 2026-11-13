// Package thumbs membuat thumbnail/preview JPEG dengan tool eksternal
// (vipsthumbnail, sips di macOS, ffmpeg) dan menyimpannya di cache disk.
//
// Tool eksternal dipilih agar tanpa cgo dan HEIC/video bisa ditangani; hanya
// `workers` proses berjalan bersamaan supaya RAM perangkat kecil aman.
package thumbs

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	ThumbEdge   = 400  // sisi terpanjang thumbnail grid
	PreviewEdge = 1920 // sisi terpanjang preview lightbox
	quality     = 80
	toolTimeout = 90 * time.Second
)

// ErrBusy: antrean thumbnail penuh (lindungi server dari banjir permintaan); klien boleh mencoba lagi.
var ErrBusy = errors.New("thumbnail queue busy")

// maxQueue: batas permintaan pengguna yang berjalan + menunggu.
const maxQueue = 96

// ErrNoTool dikembalikan bila tidak ada tool yang cocok untuk tipe file.
var ErrNoTool = errors.New("no thumbnail tool available for this file type")

type tool struct {
	name           string
	images, videos bool
	run            func(ctx context.Context, src, dst string, edge int, video bool) error
}

// Generator membuat dan menyimpan thumbnail di Root.
type Generator struct {
	root  string
	sem   chan struct{} // slot untuk pekerjaan latar belakang (warm-up)
	fg    chan struct{} // slot terpisah untuk request pengguna agar tidak antre di belakang warm-up
	fgN   atomic.Int32  // jumlah request pengguna yang sedang menunggu/berjalan
	tools []tool
}

type bgKey struct{}

// Background menandai ctx sebagai pekerjaan latar belakang berprioritas rendah.
func Background(ctx context.Context) context.Context { return context.WithValue(ctx, bgKey{}, true) }

// acquire mengambil slot. Pekerjaan latar belakang mengalah selama ada request pengguna.
func (g *Generator) acquire(ctx context.Context) (release func(), err error) {
	if ctx.Value(bgKey{}) == nil {
		if g.fgN.Add(1) > maxQueue {
			g.fgN.Add(-1)
			return nil, ErrBusy
		}
		select {
		case g.fg <- struct{}{}:
			return func() { <-g.fg; g.fgN.Add(-1) }, nil
		case <-ctx.Done():
			g.fgN.Add(-1)
			return nil, ctx.Err()
		}
	}
	for g.fgN.Load() > 0 {
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	select {
	case g.sem <- struct{}{}:
		return func() { <-g.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// New mendeteksi tool yang tersedia. workers minimal 1.
func New(root string, workers int) *Generator {
	if workers < 1 {
		workers = 1
	}
	g := &Generator{root: root, sem: make(chan struct{}, workers), fg: make(chan struct{}, workers)}
	if p, err := exec.LookPath("vipsthumbnail"); err == nil {
		g.tools = append(g.tools, tool{"vipsthumbnail", true, false, vips(p)})
	}
	if p, err := exec.LookPath("sips"); err == nil {
		g.tools = append(g.tools, tool{"sips", true, false, sips(p)})
	}
	if p, err := exec.LookPath("heif-convert"); err == nil {
		g.tools = append(g.tools, tool{"heif-convert", true, false, heifConvert(p)})
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		g.tools = append(g.tools, tool{"ffmpeg", true, true, ffmpeg(p)})
	}
	// purego sebagai fallback universal untuk image JPEG, PNG, GIF
	g.tools = append(g.tools, tool{"purego", true, false, pureGo})
	return g
}

// Tools mengembalikan nama tool yang terdeteksi (urut prioritas).
func (g *Generator) Tools() []string {
	names := make([]string, len(g.tools))
	for i, t := range g.tools {
		names[i] = t.name
	}
	return names
}

// Path lokasi cache untuk (kind, id), mis. <root>/thumb/0a/10.jpg.
func (g *Generator) Path(kind string, id int64) string {
	return filepath.Join(g.root, kind, fmt.Sprintf("%02x", id%256), strconv.FormatInt(id, 10)+".jpg")
}

// Ensure memastikan cache (kind,id) ada dan lebih baru dari mtime sumber.
// generated=true bila file baru saja dibuat.
func (g *Generator) Ensure(ctx context.Context, kind string, id int64, src string, video bool, mtime int64, edge int) (dst string, generated bool, err error) {
	dst = g.Path(kind, id)
	if fresh(dst, mtime) {
		return dst, false, nil
	}
	release, err := g.acquire(ctx)
	if err != nil {
		return "", false, err
	}
	defer release()
	if fresh(dst, mtime) { // dibuat request lain saat menunggu
		return dst, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", false, err
	}
	tmp := strings.TrimSuffix(dst, ".jpg") + ".part.jpg"
	defer os.Remove(tmp)

	var errs []string
	for _, t := range g.tools {
		if (video && !t.videos) || (!video && !t.images) {
			continue
		}
		rctx, cancel := context.WithTimeout(ctx, toolTimeout)
		err := t.run(rctx, src, tmp, edge, video)
		cancel()
		if err == nil {
			if st, serr := os.Stat(tmp); serr == nil && st.Size() > 0 {
				if err := os.Rename(tmp, dst); err != nil {
					return "", false, err
				}
				return dst, true, nil
			}
			err = errors.New("empty output")
		}
		errs = append(errs, t.name+": "+err.Error())
	}
	if len(errs) == 0 {
		return "", false, ErrNoTool
	}
	return "", false, errors.New(strings.Join(errs, "; "))
}

func fresh(p string, mtime int64) bool {
	st, err := os.Stat(p)
	return err == nil && st.Size() > 0 && st.ModTime().Unix() >= mtime
}

// Dims membaca lebar/tinggi JPEG.
func Dims(p string) (w, h int, err error) {
	f, err := os.Open(p)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	return cfg.Width, cfg.Height, err
}

func runCmd(ctx context.Context, name string, args ...string) error {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

func vips(bin string) func(context.Context, string, string, int, bool) error {
	return func(ctx context.Context, src, dst string, edge int, _ bool) error {
		box := fmt.Sprintf("%dx%d>", edge, edge) // hanya memperkecil
		// Buang EXIF/GPS dari derivatif (privasi saat dibagikan), pertahankan profil warna ICC.
		// libvips < 8.15 tidak mengenal keep=icc -> pakai strip.
		err := runCmd(ctx, bin, src, "-s", box, "-o", fmt.Sprintf("%s[Q=%d,keep=icc]", dst, quality))
		if err != nil && ctx.Err() == nil {
			err = runCmd(ctx, bin, src, "-s", box, "-o", fmt.Sprintf("%s[Q=%d,strip]", dst, quality))
		}
		return err
	}
}

func sips(bin string) func(context.Context, string, string, int, bool) error {
	return func(ctx context.Context, src, dst string, edge int, _ bool) error {
		return runCmd(ctx, bin, "-Z", strconv.Itoa(edge), "-s", "format", "jpeg",
			"-s", "formatOptions", strconv.Itoa(quality), src, "--out", dst)
	}
}

func ffmpeg(bin string) func(context.Context, string, string, int, bool) error {
	return func(ctx context.Context, src, dst string, edge int, video bool) error {
		vf := fmt.Sprintf("scale=w='min(%d,iw)':h='min(%d,ih)':force_original_aspect_ratio=decrease", edge, edge)
		seeks := [][]string{nil}
		if video {
			seeks = [][]string{{"-ss", "1"}, nil} // lewati frame hitam; ulangi dari awal bila video pendek
		}
		var err error
		for _, seek := range seeks {
			args := append([]string{"-nostdin", "-v", "error", "-y"}, seek...)
			args = append(args, "-i", src, "-frames:v", "1", "-vf", vf, "-q:v", "4", "-update", "1", dst)
			if err = runCmd(ctx, bin, args...); err == nil {
				if st, serr := os.Stat(dst); serr == nil && st.Size() > 0 {
					return nil
				}
			}
			// Fallback bila filtergraph bentrok (mis. HEIC tile-grid dengan internal complex filtergraph)
			if err != nil && strings.Contains(err.Error(), "complex filtergraph") {
				argsAlt := append([]string{"-nostdin", "-v", "error", "-y"}, seek...)
				argsAlt = append(argsAlt, "-i", src, "-frames:v", "1", "-s", fmt.Sprintf("%dx%d", edge, edge), "-q:v", "4", "-update", "1", dst)
				if errAlt := runCmd(ctx, bin, argsAlt...); errAlt == nil {
					if st, serr := os.Stat(dst); serr == nil && st.Size() > 0 {
						return nil
					}
				}
			}
		}
		if err == nil {
			err = errors.New("no frame")
		}
		return err
	}
}

func isHEIF(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".heic" || ext == ".heif" || ext == ".hif"
}

func heifConvert(bin string) func(context.Context, string, string, int, bool) error {
	return func(ctx context.Context, src, dst string, edge int, video bool) error {
		if video || !isHEIF(src) {
			return errors.New("not a heif file")
		}
		rawJpg := dst + ".raw.jpg"
		defer os.Remove(rawJpg)
		cmd := exec.CommandContext(ctx, bin, "-q", "85", src, rawJpg)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("heif-convert: %w: %s", err, strings.TrimSpace(string(out)))
		}
		if err := pureGo(ctx, rawJpg, dst, edge, false); err != nil {
			return os.Rename(rawJpg, dst)
		}
		return nil
	}
}

func pureGo(ctx context.Context, src, dst string, edge int, _ bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return errors.New("invalid image dimensions")
	}

	targetW, targetH := w, h
	if w > edge || h > edge {
		if w >= h {
			targetW = edge
			targetH = (h * edge) / w
		} else {
			targetH = edge
			targetW = (w * edge) / h
		}
	}
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}

	resized := scaleImage(img, targetW, targetH)
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, resized, &jpeg.Options{Quality: quality})
}

func scaleImage(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	for y := range h {
		sy := bounds.Min.Y + (y*srcH)/h
		for x := range w {
			sx := bounds.Min.X + (x*srcW)/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
