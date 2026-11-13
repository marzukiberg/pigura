// Metadata detail aset (EXIF, GPS, codec) untuk panel info di UI pemilik.
package server

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/marzukiberg/pigura/internal/meta"
)

// metaCache: hasil dibaca sekali per (id, mtime); dibatasi agar memori tak tumbuh tanpa batas.
type metaCache struct {
	mu sync.Mutex
	m  map[string]*meta.Info
}

const metaCacheMax = 512

func (c *metaCache) get(k string) (*meta.Info, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	return v, ok
}

func (c *metaCache) put(k string, v *meta.Info) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || len(c.m) >= metaCacheMax {
		c.m = map[string]*meta.Info{}
	}
	c.m[k] = v
}

// metaReader memilih pembaca: bisa diganti di test lewat Server.Meta.
func (s *Server) metaReader() *meta.Reader {
	if s.Meta != nil {
		return s.Meta
	}
	return meta.DefaultReader
}

// GET /api/assets/{id}/metadata — hanya pemilik (bukan rute berbagi, jadi GPS/EXIF tidak terbuka untuk pengunjung).
func (s *Server) assetMetadata(w http.ResponseWriter, r *http.Request) {
	a, full, ok := s.lookup(w, r)
	if !ok {
		return
	}
	key := strconv.FormatInt(a.ID, 10) + ":" + strconv.FormatInt(a.Mtime, 10)
	info, hit := s.metaCache.get(key)
	if !hit {
		var err error
		info, err = s.readMeta(r.Context(), full, a.Type)
		if err != nil {
			log.Printf("metadata %d (%s): %v", a.ID, a.Rel, err)
			// tetap 200 dengan info minimal agar panel tidak error; detail EXIF hanya tidak tersedia
			info = &meta.Info{Kind: a.Type}
		} else {
			s.metaCache.put(key, info)
		}
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) readMeta(ctx context.Context, path, kind string) (*meta.Info, error) {
	return s.metaReader().Read(ctx, path, kind)
}
