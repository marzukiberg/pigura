// Pemindaian berkala: file baru yang masuk ke library (mis. hasil sinkronisasi Syncthing dari
// ponsel) langsung terindeks tanpa menunggu pemindaian manual.
package server

import (
	"context"
	"errors"
	"log"
	"time"
)

// minAutoScan: batas bawah interval agar HDD tidak terus-menerus ditelusuri.
const minAutoScan = 30 * time.Second

// StartAutoScan memindai library tiap `every` (<=0 = nonaktif) sampai ctx selesai.
// Pemindaian bersifat tunggal: bila pemindaian lain sedang berjalan, putaran ini dilewati.
func (s *Server) StartAutoScan(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	if every < minAutoScan {
		every = minAutoScan
	}
	go s.autoScanLoop(ctx, every)
}

func (s *Server) autoScanLoop(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.autoScanOnce(ctx)
		}
	}
}

func (s *Server) autoScanOnce(ctx context.Context) {
	if s.Library == "" {
		return
	}
	res, err := s.RunScan(ctx)
	switch {
	case errors.Is(err, ErrScanRunning), errors.Is(err, context.Canceled):
	case err != nil:
		log.Printf("auto-scan: %v", err)
	case res.Changed > 0:
		log.Printf("auto-scan: %d file baru/berubah (dilihat %d)", res.Changed, res.Seen)
	}
}
