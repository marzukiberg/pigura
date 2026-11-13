package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/marzukiberg/pigura/internal/db"
	"github.com/marzukiberg/pigura/internal/scanner"
	"github.com/marzukiberg/pigura/internal/server"
	"github.com/marzukiberg/pigura/internal/thumbs"
	"github.com/marzukiberg/pigura/internal/web"
)

// version diisi saat build: -ldflags "-X main.version=1.2.3" (lihat Makefile / Dockerfile / release.yml).
var version = "dev"

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// staticFS: frontend dari folder (bila diberikan) dengan cadangan versi tertanam.
func staticFS(dir string) fs.FS {
	if dir == "" {
		return web.FS()
	}
	log.Printf("frontend dari folder %s (cadangan: tertanam)", dir)
	return web.Overlay(os.DirFS(dir), web.FS())
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		log.Printf("%s=%q bukan durasi valid; dipakai %s", k, v, def)
	}
	return def
}

func main() {
	addr := flag.String("addr", env("PIGURA_ADDR", "127.0.0.1:8080"), "alamat listen")
	dbPath := flag.String("db", env("PIGURA_DB", "./data/pigura.db"), "path SQLite")
	library := flag.String("library", env("PIGURA_LIBRARY", ""), "root library (read-only; disimpan di DB)")
	scanBase := flag.String("scan-base", env("PIGURA_SCAN_BASE", "/srv/photos"), "base folder yang diizinkan untuk scan")
	cache := flag.String("cache", env("PIGURA_CACHE", ""), "direktori cache thumbnail (default: <dir db>/cache)")
	workers, _ := strconv.Atoi(env("PIGURA_THUMB_WORKERS", "1"))
	flag.IntVar(&workers, "thumb-workers", workers, "jumlah proses thumbnail paralel")
	staticDir := flag.String("static", env("PIGURA_STATIC_DIR", ""), "folder frontend hasil build; menimpa yang tertanam di binary (kosong = hanya tertanam)")
	scan := flag.Bool("scan", false, "scan library saat start")
	scanEvery := flag.Duration("scan-interval", envDuration("PIGURA_SCAN_INTERVAL", 0), "pindai library berkala (mis. 5m); 0 = nonaktif")
	showVersion := flag.Bool("version", false, "tampilkan versi lalu keluar")
	flag.Parse()
	if *showVersion {
		fmt.Println("pigura", version)
		return
	}
	log.Printf("pigura %s", version)

	d, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer d.Close()

	// Prioritas konfigurasi library: flag/env baru, nilai tersimpan di SQLite,
	// lalu deteksi layout Immich standar di bawah scan-base. Semua kandidat
	// wajib menjadi anak dari scan-base; root disk tidak pernah dipindai.
	if *library != "" {
		root, err := scanner.ValidateRoot(*library, *scanBase)
		if err != nil {
			log.Fatalf("library tidak aman: %v", err)
		}
		if err := db.SetSetting(d, db.LibraryRootKey, root); err != nil {
			log.Fatalf("simpan library config: %v", err)
		}
		*library = root
	} else if saved, err := db.Setting(d, db.LibraryRootKey); err != nil {
		log.Fatalf("baca library config: %v", err)
	} else if saved != "" {
		root, err := scanner.ValidateRoot(saved, *scanBase)
		if err != nil {
			log.Fatalf("library config DB tidak aman: %v", err)
		}
		*library = root
	} else if root, err := scanner.FindImmichLibrary(*scanBase); err == nil {
		if err := db.SetSetting(d, db.LibraryRootKey, root); err != nil {
			log.Fatalf("simpan library terdeteksi: %v", err)
		}
		*library = root
		log.Printf("library Immich terdeteksi: %s", root)
	} else {
		log.Printf("library belum dikonfigurasi: %v", err)
	}

	if *cache == "" {
		*cache = filepath.Join(filepath.Dir(*dbPath), "cache")
	}
	gen := thumbs.New(*cache, workers)
	if len(gen.Tools()) == 0 {
		log.Print("PERINGATAN: tidak ada tool thumbnail (vipsthumbnail/sips/ffmpeg); thumbnail tidak akan dibuat")
	} else {
		log.Printf("tool thumbnail: %v, cache=%s", gen.Tools(), *cache)
	}

	s := &server.Server{Version: version, DB: d, Library: *library, ScanBase: *scanBase, Gen: gen, Static: staticFS(*staticDir),
		// Keamanan: dibaca dari env agar rahasia tidak muncul di daftar proses.
		Token:          os.Getenv("PIGURA_TOKEN"),
		AllowedHosts:   server.SplitList(os.Getenv("PIGURA_ALLOWED_HOSTS")),
		AllowedOrigins: server.SplitList(os.Getenv("PIGURA_ALLOWED_ORIGINS")),
	}
	if s.Token != "" {
		log.Print("akses dilindungi token (PIGURA_TOKEN aktif)")
	}
	s.PublicURL = strings.TrimRight(os.Getenv("PIGURA_PUBLIC_URL"), "/")
	s.PublicHosts = server.SplitList(os.Getenv("PIGURA_PUBLIC_HOSTS"))

	if *scan {
		if *library == "" {
			log.Fatal("-scan butuh -library")
		}
		start := time.Now()
		res, err := s.RunScan(context.Background())
		if err != nil {
			log.Fatalf("scan: %v", err)
		}
		log.Printf("scan selesai: seen=%d changed=%d (%s)", res.Seen, res.Changed, time.Since(start).Round(time.Millisecond))
	}
	// Thumbnail dibuat sesuai permintaan (hanya tile yang tampil di viewport).

	srv := &http.Server{
		Addr:              *addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second, // permintaan kecil; respons (video) tidak dibatasi WriteTimeout
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}

	// Listener publik (opsional): hanya tautan berbagi. Arahkan tunnel/Funnel ke sini, bukan ke -addr.
	var pubSrv *http.Server
	if pa := os.Getenv("PIGURA_PUBLIC_ADDR"); pa != "" {
		if h, _, err := net.SplitHostPort(pa); err != nil || (h != "127.0.0.1" && h != "localhost" && h != "::1") {
			log.Printf("PERINGATAN: PIGURA_PUBLIC_ADDR=%s bukan loopback; sebaiknya bind 127.0.0.1 dan taruh TLS/proxy di depannya", pa)
		}
		pubSrv = &http.Server{
			Addr:              pa,
			Handler:           s.PublicHandler(),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    32 << 10,
		}
		go func() {
			log.Printf("listener publik (hanya tautan berbagi) di %s", pa)
			if err := pubSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("listener publik: %v", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *scanEvery > 0 {
		log.Printf("pemindaian berkala tiap %s", *scanEvery)
		s.StartAutoScan(ctx, *scanEvery)
	}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if pubSrv != nil {
			pubSrv.Shutdown(shCtx)
		}
		srv.Shutdown(shCtx)
	}()

	log.Printf("pigura listen %s (library=%q)", *addr, *library)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
