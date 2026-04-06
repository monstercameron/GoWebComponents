BEGIN TRANSACTION;

DROP TABLE IF EXISTS docs_run;
DROP TABLE IF EXISTS docs_node;
DROP TABLE IF EXISTS docs_document;

CREATE TABLE docs_document (
  id INTEGER PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('md','html','json','other')),
  status TEXT NOT NULL DEFAULT 'draft',
  canonical_url TEXT,
  meta_json TEXT NOT NULL DEFAULT '{}',
  created_at_utc TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at_utc TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE docs_node (
  id INTEGER PRIMARY KEY,
  document_id INTEGER NOT NULL,
  node_type TEXT NOT NULL CHECK (node_type IN ('title','heading','section','paragraph','bullet','checklist','api','link','code','table','html','asset','note')),
  level INTEGER NOT NULL DEFAULT 1,
  title TEXT,
  body_text TEXT,
  payload_json TEXT NOT NULL DEFAULT '{}',
  sort_order INTEGER NOT NULL DEFAULT 0,
  section_anchor TEXT,
  source_line_no INTEGER,
  created_at_utc TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(document_id) REFERENCES docs_document(id) ON DELETE CASCADE
);

CREATE TABLE docs_run (
  id INTEGER PRIMARY KEY,
  document_id INTEGER NOT NULL,
  source_kind TEXT NOT NULL CHECK (source_kind IN ('benchmark_json','raw_payload','asset_snapshot','imported_payload')),
  run_key TEXT NOT NULL,
  source_path TEXT NOT NULL,
  generated_at_utc TEXT,
  run_meta_json TEXT NOT NULL DEFAULT '{}',
  payload_json TEXT NOT NULL,
  created_at_utc TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(document_id, run_key),
  FOREIGN KEY(document_id) REFERENCES docs_document(id) ON DELETE CASCADE
);

CREATE INDEX idx_docs_node_doc_type_order ON docs_node(document_id, node_type, sort_order);
CREATE INDEX idx_docs_node_section_anchor ON docs_node(document_id, section_anchor);
CREATE INDEX idx_docs_run_key ON docs_run(run_key);
CREATE UNIQUE INDEX idx_docs_run_doc_kind_path ON docs_run(document_id, source_kind, source_path);

COMMIT;
