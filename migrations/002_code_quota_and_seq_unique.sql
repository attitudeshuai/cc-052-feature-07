BEGIN;

-- Per-batch code quota (NULL = unlimited). Set at batch creation, used to
-- reject generation requests that would exceed the remaining quota.
ALTER TABLE crop_batch ADD COLUMN IF NOT EXISTS code_quota INT;

-- Hard guarantee that one seq segment is issued at most once per batch.
-- NOTE: if legacy data already contains duplicate (batch_id, seq) rows,
-- this index will fail to build; clean up duplicates first, e.g.:
--   DELETE FROM trace_code a USING trace_code b
--   WHERE a.batch_id = b.batch_id AND a.seq = b.seq AND a.id > b.id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_trace_code_batch_seq ON trace_code(batch_id, seq);

COMMIT;
