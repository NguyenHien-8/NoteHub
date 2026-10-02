package backend

const schemaSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS app_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS memo (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  search_text TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_memo_timeline
  ON memo(created_ts DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_memo_updated
  ON memo(updated_ts DESC, id DESC);

CREATE TABLE IF NOT EXISTS memo_tag (
  memo_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  tag_norm TEXT NOT NULL,
  PRIMARY KEY (memo_id, tag_norm),
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_memo_tag_norm
  ON memo_tag(tag_norm, memo_id);

CREATE TABLE IF NOT EXISTS attachment (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  memo_id INTEGER NOT NULL,
  created_ts INTEGER NOT NULL,
  filename TEXT NOT NULL,
  mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
  size INTEGER NOT NULL DEFAULT 0,
  sha256 TEXT NOT NULL,
  relative_path TEXT NOT NULL UNIQUE,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_attachment_memo
  ON attachment(memo_id, created_ts ASC, id ASC);

CREATE TABLE IF NOT EXISTS memo_share (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token TEXT NOT NULL UNIQUE,
  memo_id INTEGER NOT NULL,
  created_ts INTEGER NOT NULL,
  expires_ts INTEGER DEFAULT NULL,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_memo_share_memo
  ON memo_share(memo_id);
`
