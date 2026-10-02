package sqlite

const schemaV1 = `
CREATE TABLE IF NOT EXISTS memo (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  content TEXT NOT NULL DEFAULT '',
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  revision INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1)
);

CREATE INDEX IF NOT EXISTS idx_memo_timeline
  ON memo(created_ts DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_memo_updated
  ON memo(updated_ts DESC, id DESC);

CREATE TABLE IF NOT EXISTS memo_tag (
  memo_id INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  tag_norm TEXT NOT NULL,
  PRIMARY KEY (memo_id, tag_norm)
);
CREATE INDEX IF NOT EXISTS idx_memo_tag_norm
  ON memo_tag(tag_norm, memo_id);

CREATE TABLE IF NOT EXISTS attachment (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  memo_id INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
  created_ts INTEGER NOT NULL,
  filename TEXT NOT NULL,
  mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
  size INTEGER NOT NULL CHECK (size >= 0),
  sha256 TEXT NOT NULL,
  relative_path TEXT NOT NULL UNIQUE
);
CREATE INDEX IF NOT EXISTS idx_attachment_memo
  ON attachment(memo_id, created_ts ASC, id ASC);

CREATE TABLE IF NOT EXISTS memo_share (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  memo_id INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
  token_hash BLOB NOT NULL UNIQUE,
  created_ts INTEGER NOT NULL,
  expires_ts INTEGER DEFAULT NULL
);
CREATE INDEX IF NOT EXISTS idx_memo_share_memo
  ON memo_share(memo_id, created_ts DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_memo_share_expiry
  ON memo_share(expires_ts) WHERE expires_ts IS NOT NULL;

-- FTS5 + unicode61 gives substantially better multilingual search than
-- SQLite's ASCII-only NOCASE/LOWER behavior. NoteHub duplicates only memo text
-- in the FTS index; SQLite remains the source of truth.
CREATE VIRTUAL TABLE IF NOT EXISTS memo_fts USING fts5(
  content,
  tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TRIGGER IF NOT EXISTS memo_fts_ai AFTER INSERT ON memo BEGIN
  INSERT INTO memo_fts(rowid, content) VALUES (new.id, new.content);
END;
CREATE TRIGGER IF NOT EXISTS memo_fts_ad AFTER DELETE ON memo BEGIN
  DELETE FROM memo_fts WHERE rowid = old.id;
END;
CREATE TRIGGER IF NOT EXISTS memo_fts_au AFTER UPDATE OF content ON memo BEGIN
  UPDATE memo_fts SET content = new.content WHERE rowid = new.id;
END;
`
