// Lapisan keamanan HTTP: host allowlist, same-origin, header keamanan, batas Range.
package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// secure membungkus handler privat (API penuh) dengan seluruh pemeriksaan di atas.
func (s *Server) secure(next http.Handler) http.Handler { return s.secureWith(next, false) }

// securePublic membungkus listener publik: hanya rute berbagi, tanpa token, host dari PublicHosts.
func (s *Server) securePublic(next http.Handler) http.Handler { return s.secureWith(next, true) }

func (s *Server) secureWith(next http.Handler, pub bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.setSecurityHeaders(w)
		if pub {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
		}

		// Listener publik: tidak ada otoritas ambient (hanya tautan berkunci acak), jadi DNS rebinding
		// tak relevan; tanpa PublicHosts semua host diterima, dengan PublicHosts hanya yang terdaftar.
		if pub {
			if len(s.PublicHosts) > 0 && !hostIn(r.Host, s.PublicHosts) {
				httpErr(w, http.StatusMisdirectedRequest, "host not allowed")
				return
			}
		} else if !s.hostAllowed(r.Host) {
			httpErr(w, http.StatusMisdirectedRequest, "host not allowed")
			return
		}

		if o := r.Header.Get("Origin"); o != "" {
			switch {
			case sameOrigin(o, r.Host):
				// ok
			case s.originAllowed(o):
				w.Header().Set("Access-Control-Allow-Origin", o)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
				if r.Method == http.MethodOptions {
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range, Authorization")
					w.Header().Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			default:
				httpErr(w, http.StatusForbidden, "cross-origin request blocked")
				return
			}
		}

		// Batasi jumlah rentang pada header Range (celah DoS pada parser Range).
		if rg := r.Header.Get("Range"); rg != "" && (len(rg) > 256 || strings.Count(rg, ",") > 4) {
			httpErr(w, http.StatusRequestedRangeNotSatisfiable, "too many ranges")
			return
		}

		if !pub && s.Token != "" && strings.HasPrefix(r.URL.Path, "/api/") && !authExempt(r) && !s.authed(r) {
			httpErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hostIn(hostport string, list []string) bool {
	h := splitHost(hostport)
	for _, a := range list {
		if strings.EqualFold(a, h) {
			return true
		}
	}
	return false
}

func (s *Server) setSecurityHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	connect := "'self'"
	for _, o := range s.AllowedOrigins {
		connect += " " + o
	}
	h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
		"img-src 'self' data: blob:; media-src 'self' blob:; font-src 'self' data:; connect-src "+connect+"; "+
		"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
	if len(s.AllowedOrigins) == 0 {
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
	}
}

// hostAllowed: loopback, IP literal, nama tanpa titik (MagicDNS pendek), *.ts.net,
// atau yang didaftarkan. Domain publik asing (vektor DNS rebinding) ditolak.
func (s *Server) hostAllowed(hostport string) bool {
	h := splitHost(hostport)
	switch {
	case h == "":
		return false
	case net.ParseIP(h) != nil, h == "localhost", !strings.Contains(h, "."),
		strings.HasSuffix(h, ".ts.net"), strings.HasSuffix(h, ".localhost"):
		return true
	}
	for _, a := range s.AllowedHosts {
		if strings.EqualFold(a, h) {
			return true
		}
	}
	return false
}

func sameOrigin(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, host)
}

func (s *Server) originAllowed(origin string) bool {
	for _, a := range s.AllowedOrigins {
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}
