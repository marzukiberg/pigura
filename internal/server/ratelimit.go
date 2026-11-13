// Pembatas laju sederhana (jendela geser, dalam memori) dan penentuan IP klien.
package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter: jendela geser per kunci, dalam memori. Dipakai untuk akses publik, percobaan
// membuka password tautan, dan kegagalan login pemilik.
type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

// recent mengembalikan kejadian kunci dalam window (sambil membuang yang usang). l.mu harus dipegang.
func (l *rateLimiter) recent(key string, window time.Duration) []time.Time {
	if l.hits == nil {
		l.hits = map[string][]time.Time{}
	}
	cut := time.Now().Add(-window)
	keep := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	l.hits[key] = keep
	if len(l.hits) > 10000 { // pangkas kunci usang agar memori tak tumbuh tanpa batas
		for k, v := range l.hits {
			if len(v) == 0 || v[len(v)-1].Before(cut) {
				delete(l.hits, k)
			}
		}
	}
	return keep
}

// allow mencatat satu percobaan; false bila melewati max dalam window.
func (l *rateLimiter) allow(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hits[key] = append(l.recent(key, window), time.Now())
	return len(l.hits[key]) <= max
}

// blocked: true bila sudah ada >= max kejadian dalam window (tanpa mencatat apa pun).
func (l *rateLimiter) blocked(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key, window)) >= max
}

// record mencatat satu kejadian (mis. satu kegagalan login).
func (l *rateLimiter) record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hits == nil {
		l.hits = map[string][]time.Time{}
	}
	l.hits[key] = append(l.hits[key], time.Now())
}

// clientIP: header proxy hanya dipercaya bila koneksi datang dari loopback (cloudflared/proxy lokal).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if v := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); v != "" {
			return v
		}
		if v := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); v != "" {
			return v
		}
	}
	return host
}
