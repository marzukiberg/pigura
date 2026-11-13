CREATE TABLE IF NOT EXISTS asset (
  id         INTEGER PRIMARY KEY,
  path       TEXT    NOT NULL UNIQUE,      -- relatif terhadap library root
  type       TEXT    NOT NULL,             -- 'image' | 'video'
  sha1       TEXT,                         -- hash isi penuh; hanya diisi untuk kandidat duplikat
  qhash      TEXT,                         -- hash sampel (awal+tengah+akhir)
  taken_at   INTEGER NOT NULL,             -- unix detik
  width      INTEGER NOT NULL DEFAULT 0,
  height     INTEGER NOT NULL DEFAULT 0,
  size       INTEGER NOT NULL DEFAULT 0,
  mtime      INTEGER NOT NULL DEFAULT 0,
  duration   REAL    NOT NULL DEFAULT 0,
  phash      INTEGER,
  thumb_path TEXT,                         -- relatif terhadap thumbs root
  deleted_at INTEGER
);
CREATE INDEX IF NOT EXISTS idx_asset_taken ON asset(taken_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_asset_size  ON asset(size);
CREATE INDEX IF NOT EXISTS idx_asset_sha1  ON asset(sha1) WHERE sha1 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_phash ON asset(phash) WHERE phash IS NOT NULL;

-- Konfigurasi server yang harus bertahan saat binary/service direstart.
-- library_root selalu divalidasi aplikasi sebelum dipakai untuk scan.
CREATE TABLE IF NOT EXISTS setting (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS album (
  id         INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  created_at INTEGER NOT NULL,
  cover_id   INTEGER REFERENCES asset(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS album_asset (
  album_id INTEGER NOT NULL REFERENCES album(id) ON DELETE CASCADE,
  asset_id INTEGER NOT NULL REFERENCES asset(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (album_id, asset_id)
);

CREATE TABLE IF NOT EXISTS dupe_group (
  id   INTEGER PRIMARY KEY,
  kind TEXT NOT NULL                       -- 'exact' | 'similar'
);

CREATE TABLE IF NOT EXISTS dupe_member (
  group_id INTEGER NOT NULL REFERENCES dupe_group(id) ON DELETE CASCADE,
  asset_id INTEGER NOT NULL REFERENCES asset(id) ON DELETE CASCADE,
  PRIMARY KEY (group_id, asset_id)
);

-- Tautan berbagi (publik, dilindungi kunci acak; opsional password & kedaluwarsa).
CREATE TABLE IF NOT EXISTS share (
  id             INTEGER PRIMARY KEY,
  key            TEXT    NOT NULL UNIQUE,   -- 22 karakter base64url (128 bit acak)
  kind           TEXT    NOT NULL,          -- 'asset' | 'album'
  target_id      INTEGER NOT NULL,
  title          TEXT    NOT NULL DEFAULT '',
  created_at     INTEGER NOT NULL,
  expires_at     INTEGER,                   -- NULL = tidak kedaluwarsa
  pw             TEXT,                      -- NULL = tanpa password; "p1$iter$salt$hash" (PBKDF2-SHA256)
  allow_download INTEGER NOT NULL DEFAULT 0,
  views          INTEGER NOT NULL DEFAULT 0,
  last_viewed_at INTEGER,
  revoked_at     INTEGER
);
CREATE INDEX IF NOT EXISTS idx_share_target ON share(kind, target_id);
