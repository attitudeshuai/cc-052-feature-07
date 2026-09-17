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

func (r *TraceCodeRepo) BatchInsert(codes []model.TraceCode) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tc := range codes {
		if _, err := stmt.Exec(tc.BatchID, tc.Code, tc.Seq); err != nil {
			return err
		}
	}

	return tx.Commit()
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

func (r *TraceCodeRepo) GetMaxSeqByBatch(batchID int64) (int, error) {
	var maxSeq int
	query := `SELECT COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`
	if err := r.db.Get(&maxSeq, query, batchID); err != nil {
		return 0, err
	}
	return maxSeq, nil
}

func (r *TraceCodeRepo) CountByBatch(batchID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM trace_code WHERE batch_id = $1`
	if err := r.db.Get(&count, query, batchID); err != nil {
		return 0, err
	}
	return count, nil
}