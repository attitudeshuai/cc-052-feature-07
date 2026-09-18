package repository

import (
	"cc-052/internal/model"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type TraceCodeRepo struct {
	db *sqlx.DB
}

func NewTraceCodeRepo(db *sqlx.DB) *TraceCodeRepo {
	return &TraceCodeRepo{db: db}
}

// IssueCodes 在单个事务内完成「锁批次行 → 取当前最大序号 → 生成并插入」。
// crop_batch 行上的 FOR UPDATE 锁让同一批次的并发发码串行执行：
// 后到的请求在锁上等待，提交后读到的 maxSeq 已包含前一批的结果，
// 两个请求不会拿到同一段号。gen 按分配到的序号生成码字符串。
// inserted 是实际落库的码（以 RowsAffected 为准），skipped 是撞唯一约束被跳过的码。
func (r *TraceCodeRepo) IssueCodes(batchID int64, count int, gen func(seq int64) string) (inserted []string, skipped []string, err error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	// 串行化同一批次的发码请求
	var lockedID int64
	if err := tx.Get(&lockedID, `SELECT id FROM crop_batch WHERE id = $1 FOR UPDATE`, batchID); err != nil {
		return nil, nil, fmt.Errorf("lock batch row: %w", err)
	}

	var maxSeq int
	if err := tx.Get(&maxSeq, `SELECT COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`, batchID); err != nil {
		return nil, nil, err
	}

	stmt, err := tx.Preparex(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return nil, nil, err
	}
	defer stmt.Close()

	inserted = make([]string, 0, count)
	skipped = make([]string, 0)
	for i := 0; i < count; i++ {
		seq := maxSeq + i + 1
		code := gen(int64(seq))
		res, err := stmt.Exec(batchID, code, seq)
		if err != nil {
			return nil, nil, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted = append(inserted, code)
		} else {
			skipped = append(skipped, code)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
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

// StatsByBatch 对账：已分配序号（max_seq）应当等于落库行数（issued），
// 缺号（分配了没落下）或重号（同一序号多行）都会在这里暴露出来。
func (r *TraceCodeRepo) StatsByBatch(batchID int64) (*model.TraceCodeStats, error) {
	stats := &model.TraceCodeStats{BatchID: batchID}
	var distinctSeq int
	query := `SELECT COUNT(*), COALESCE(MAX(seq), 0), COUNT(DISTINCT seq) FROM trace_code WHERE batch_id = $1`
	if err := r.db.QueryRow(query, batchID).Scan(&stats.Issued, &stats.MaxSeq, &distinctSeq); err != nil {
		return nil, err
	}
	stats.Gaps = stats.MaxSeq - distinctSeq
	stats.DuplicateSeqs = stats.Issued - distinctSeq
	stats.Consistent = stats.Issued == stats.MaxSeq && stats.DuplicateSeqs == 0

	if stats.Gaps > 0 {
		missingQuery := `SELECT seq FROM (
				SELECT generate_series(1, $2) AS seq
			) g
			WHERE seq NOT IN (SELECT seq FROM trace_code WHERE batch_id = $1)
			ORDER BY seq LIMIT 100`
		var missing []int
		if err := r.db.Select(&missing, missingQuery, batchID, stats.MaxSeq); err != nil {
			return nil, err
		}
		stats.MissingSeqs = missing
	}
	return stats, nil
}