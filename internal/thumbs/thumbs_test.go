package thumbs

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureThumb(t *testing.T) {
	dir := t.TempDir()
	g := New(filepath.Join(dir, "cache"), 1)
	if len(g.Tools()) == 0 {
		t.Skip("tidak ada tool thumbnail (vipsthumbnail/sips/ffmpeg)")
	}

	src := filepath.Join(dir, "in.png")
	img := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for x := 0; x < 800; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	f, _ := os.Create(src)
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	dst, gen, err := g.Ensure(context.Background(), "thumb", 7, src, false, 0, ThumbEdge)
	if err != nil || !gen {
		t.Fatalf("Ensure = %q gen=%v err=%v", dst, gen, err)
	}
	w, h, err := Dims(dst)
	if err != nil || w != 400 || h != 200 {
		t.Fatalf("dims = %dx%d err=%v, want 400x200", w, h, err)
	}
	if _, gen, err = g.Ensure(context.Background(), "thumb", 7, src, false, 0, ThumbEdge); err != nil || gen {
		t.Fatalf("kedua kali harus dari cache: gen=%v err=%v", gen, err)
	}
}

func TestAcquireRejectsWhenQueueFull(t *testing.T) {
	g := New(t.TempDir(), 1)
	g.fgN.Store(maxQueue) // antrean pengguna penuh
	if _, err := g.acquire(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatalf("acquire saat penuh = %v, want ErrBusy", err)
	}
	if g.fgN.Load() != maxQueue {
		t.Fatalf("penghitung antrean bocor: %d", g.fgN.Load())
	}
	g.fgN.Store(0)
	rel, err := g.acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rel()
}
