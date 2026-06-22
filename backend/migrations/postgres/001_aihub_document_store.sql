CREATE TABLE IF NOT EXISTS aihub_document (
  namespace_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  id TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY(namespace_id, kind, id)
);
CREATE INDEX IF NOT EXISTS idx_aihub_document_kind_updated ON aihub_document(kind, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_aihub_document_payload_gin ON aihub_document USING GIN(payload);
CREATE TABLE IF NOT EXISTS aihub_sequence (
  name TEXT PRIMARY KEY,
  value BIGINT NOT NULL DEFAULT 0
);
