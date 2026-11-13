// Util HTTP bersama: respons JSON, galat, parsing parameter.
package server

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func serverErr(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	httpErr(w, http.StatusInternalServerError, "internal error")
}

func queryInt(r *http.Request, key string, def int64) int64 {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

// pathID membaca {id} dari path; menjawab 400 dan false bila bukan angka.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpErr(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// splitHost mengembalikan nama host dari "host:port" / "[v6]:port" dalam huruf kecil, tanpa titik akhir.
func splitHost(hostport string) string {
	h := hostport
	if hh, _, err := net.SplitHostPort(hostport); err == nil {
		h = hh
	}
	return strings.ToLower(strings.TrimSuffix(strings.Trim(h, "[]"), "."))
}
