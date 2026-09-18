package repository

import (
	"cc-052/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type TraceCodeRepo struct {
	db *sqlx.DB
}

func NewTraceCodeRepo(db *sqlx.DB) *TraceCodeRepo {
	return &TraceCodeRepo{db: db}
}

// Begin starts a transaction for one code-generation run. The batch row
// lock, max-seq read and insert must all happen inside this transaction,
// otherwise two concurrent requests can read the same max seq and be
// handed the same number segment.
func (r *TraceCodeRepo) Begin() (*sqlx.Tx, error) {
	return r.db.Beginx()
}

// BatchInsertTx inserts codes inside tx and reports exactly what landed:
// a row with RowsAffected == 1 was persisted, == 0 was skipped because the
// code already exists (ON CONFLICT DO NOTHING). A (batch_id, seq) conflict
// is NOT covered by the conflict target and aborts the transaction loudly.
func (r *TraceCodeRepo) BatchInsertTx(tx *sqlx.Tx, codes []model.TraceCode) (inserted []model.TraceCode, skipped []model.TraceCode, err error) {
	stmt, err := tx.Preparex(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return nil, nil, err
	}
	defer stmt.Close()

	inserted = make([]model.TraceCode, 0, len(codes))
	skipped = make([]model.TraceCode, 0)
	for _, tc := range codes {
		res, err := stmt.Exec(tc.BatchID, tc.Code, tc.Seq)
		if err != nil {
			return nil, nil, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, nil, err
		}
		if affected == 0 {
			skipped = append(skipped, tc)
		} else {
			inserted = append(inserted, tc)
		}
	}
	return inserted, skipped, nil
}

func (r *TraceCodeRepo) GetByCode(code string) (*model.TraceCode, error) {
	var tc model.TraceCode
	query := `SELECT id, batch_id, code, seq, printed_at, first_scanned_at, first_scan_region, created_at
	          FROM trace_code WHERE code = $1`
	if err := r.db.Get(&tc, query, code); err != nil {
		return nil, err
	}
	return &tc, nil
}

func (r *TraceCodeRepo) MarkScanned(id int64, region string) error {
	now := time.Now()
	query := `UPDATE trace_code SET first_scanned_at = $1, first_scan_region = $2 WHERE id = $3`
	_, err := r.db.Exec(query, now, region, id)
	return err
}

// GetMaxSeqByBatchTx reads the max seq inside the generation transaction
// (after the batch row lock is held), so it never observes a stale value.
func (r *TraceCodeRepo) GetMaxSeqByBatchTx(tx *sqlx.Tx, batchID int64) (int, error) {
	var maxSeq int
	query := `SELECT COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`
	if err := tx.Get(&maxSeq, query, batchID); err != nil {
		return 0, err
	}
	return maxSeq, nil
}

// CountByBatchTx counts persisted codes inside the generation transaction;
// used for the quota check while the batch row lock is held.
func (r *TraceCodeRepo) CountByBatchTx(tx *sqlx.Tx, batchID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM trace_code WHERE batch_id = $1`
	if err := tx.Get(&count, query, batchID); err != nil {
		return 0, err
	}
	return count, nil
}

// GetSeqStats returns the reconciliation view for one batch: how many codes
// are actually persisted, the highest seq consumed, and (when the two
// disagree) up to 100 seq numbers that were consumed but never persisted.
func (r *TraceCodeRepo) GetSeqStats(batchID int64) (issued int, maxSeq int, missing []int, err error) {
	query := `SELECT COUNT(*), COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`
	if err := r.db.QueryRow(query, batchID).Scan(&issued, &maxSeq); err != nil {
		return 0, 0, nil, err
	}

	missing = make([]int, 0)
	if maxSeq > issued {
		missingQuery := `SELECT s FROM generate_series(1, $2) AS g(s)
		                 EXCEPT
		                 SELECT seq FROM trace_code WHERE batch_id = $1
		                 ORDER BY s ASC
		                 LIMIT 100`
		if err := r.db.Select(&missing, missingQuery, batchID, maxSeq); err != nil {
			return 0, 0, nil, err
		}
	}
	return issued, maxSeq, missing, nil
}
