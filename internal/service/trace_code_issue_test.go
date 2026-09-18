package service_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"cc-052/internal/model"
	"cc-052/internal/repository"
	"cc-052/internal/service"
	"cc-052/pkg/tracecode"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 集成测试需要真实 Postgres：TEST_PG_DSN=postgres://postgres@127.0.0.1:55432/codec_test?sslmode=disable
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set, skipping integration test")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// setupBatch 造一个检测合格、可发码的批次
func setupBatch(t *testing.T, db *sqlx.DB) int64 {
	t.Helper()
	var farmID, plotID, batchID int64
	if err := db.QueryRow(`INSERT INTO farm (name, region_code) VALUES ('测试农场', '110101') RETURNING id`).Scan(&farmID); err != nil {
		t.Fatalf("insert farm: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO plot (farm_id, name, area_mu) VALUES ($1, '地块1', 1.0) RETURNING id`, farmID).Scan(&plotID); err != nil {
		t.Fatalf("insert plot: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO crop_batch (plot_id, crop_id, sowing_date, expected_yield_kg) VALUES ($1, '黄瓜', '2026-01-01', 100) RETURNING id`, plotID).Scan(&batchID); err != nil {
		t.Fatalf("insert batch: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO inspection (batch_id, lab, sampled_at, result) VALUES ($1, '检测所', NOW(), 'pass')`, batchID); err != nil {
		t.Fatalf("insert inspection: %v", err)
	}
	return batchID
}

func newSvc(db *sqlx.DB) *service.TraceCodeService {
	return service.NewTraceCodeService(
		repository.NewTraceCodeRepo(db),
		repository.NewBatchRepo(db),
		repository.NewInspectionRepo(db),
		repository.NewActivityRepo(db),
		repository.NewPlotRepo(db),
		repository.NewFarmRepo(db),
	)
}

// 并发发码：同一段号只能出一份结果，返回总数必须等于落库行数
func TestGenerateCodesConcurrent(t *testing.T) {
	db := testDB(t)
	batchID := setupBatch(t, db)
	svc := newSvc(db)

	const workers = 10
	const perReq = 100

	var wg sync.WaitGroup
	results := make([]*model.GenerateCodesResult, workers)
	errs := make([]error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := svc.GenerateCodes(batchID, perReq)
			results[i] = r
			errs[i] = err
		}(w)
	}
	wg.Wait()

	total := 0
	seen := make(map[string]bool)
	for i, err := range errs {
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		r := results[i]
		if r.Count != len(r.Codes) {
			t.Fatalf("request %d: count %d != len(codes) %d", i, r.Count, len(r.Codes))
		}
		if r.Skipped != 0 {
			t.Fatalf("request %d: unexpected skipped codes %v", i, r.SkippedCodes)
		}
		for _, code := range r.Codes {
			if seen[code] {
				t.Fatalf("code %s returned by two requests", code)
			}
			seen[code] = true
		}
		total += r.Count
	}
	if want := workers * perReq; total != want {
		t.Fatalf("sum of returned counts = %d, want %d", total, want)
	}

	var rows, distinctCode, distinctSeq, maxSeq int
	err := db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT code), COUNT(DISTINCT seq), COALESCE(MAX(seq), 0)
		FROM trace_code WHERE batch_id = $1`, batchID).Scan(&rows, &distinctCode, &distinctSeq, &maxSeq)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if want := workers * perReq; rows != want || distinctCode != want || distinctSeq != want || maxSeq != want {
		t.Fatalf("db rows=%d distinctCode=%d distinctSeq=%d maxSeq=%d, want all %d", rows, distinctCode, distinctSeq, maxSeq, want)
	}

	stats, err := svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if !stats.Consistent || stats.Issued != rows || stats.MaxSeq != rows || stats.Gaps != 0 || stats.DuplicateSeqs != 0 {
		t.Fatalf("stats not consistent after concurrent issue: %+v", stats)
	}
}

// 撞重跳过：实际落库条数以 RowsAffected 为准，跳过的单独列出，且不占号段
func TestIssueCodesSkipAccounting(t *testing.T) {
	db := testDB(t)
	batchID := setupBatch(t, db)
	repo := repository.NewTraceCodeRepo(db)

	// 10 条全部生成同一个码：第 1 条落库，其余 9 条撞唯一约束被跳过
	// （code 唯一索引是全局的，常量码按批次区分保证测试可重复跑）
	fixed := fmt.Sprintf("FIX%07d", batchID)
	inserted, skipped, err := repo.IssueCodes(batchID, 10, func(seq int64) string { return fixed })
	if err != nil {
		t.Fatalf("issue codes: %v", err)
	}
	if len(inserted) != 1 || len(skipped) != 9 {
		t.Fatalf("inserted=%d skipped=%d, want 1/9", len(inserted), len(skipped))
	}

	var cnt, maxSeq int
	if err := db.QueryRow(`SELECT COUNT(*), COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`, batchID).Scan(&cnt, &maxSeq); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if cnt != 1 || maxSeq != 1 {
		t.Fatalf("db rows=%d maxSeq=%d, want 1/1", cnt, maxSeq)
	}

	// 被跳过的序号未落库，下一次发码从 2 继续
	inserted2, skipped2, err := repo.IssueCodes(batchID, 3, func(seq int64) string { return tracecode.Generate(seq) })
	if err != nil {
		t.Fatalf("issue more codes: %v", err)
	}
	if len(inserted2) != 3 || len(skipped2) != 0 {
		t.Fatalf("second issue inserted=%d skipped=%d, want 3/0", len(inserted2), len(skipped2))
	}
	if err := db.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`, batchID).Scan(&maxSeq); err != nil {
		t.Fatalf("max seq: %v", err)
	}
	if maxSeq != 4 {
		t.Fatalf("maxSeq=%d after second issue, want 4", maxSeq)
	}

	// 批次不存在时直接报错
	if _, _, err := repo.IssueCodes(999999999, 1, func(seq int64) string { return "X" }); err == nil {
		t.Fatal("expected error for missing batch, got nil")
	}
}

// 缺号与重号：对不上的时候能通过 stats 查出来
func TestStatsDetectsMismatch(t *testing.T) {
	db := testDB(t)
	batchID := setupBatch(t, db)
	repo := repository.NewTraceCodeRepo(db)
	svc := newSvc(db)

	// 预置 seq=1；发 seq 2..6 时令 seq=3 撞上已有码 → 落 2,4,5,6，缺 3
	existing := fmt.Sprintf("EX%08d", batchID)
	if _, err := db.Exec(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, 1)`, batchID, existing); err != nil {
		t.Fatalf("seed existing code: %v", err)
	}
	gen := func(seq int64) string {
		if seq == 3 {
			return existing
		}
		return fmt.Sprintf("T%06d%03d", batchID%1000000, seq)
	}
	inserted, skipped, err := repo.IssueCodes(batchID, 5, gen)
	if err != nil {
		t.Fatalf("issue codes: %v", err)
	}
	if len(inserted) != 4 || len(skipped) != 1 || skipped[0] != existing {
		t.Fatalf("inserted=%v skipped=%v, want 4 inserted and [%s] skipped", inserted, skipped, existing)
	}

	stats, err := svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.Consistent {
		t.Fatalf("stats should be inconsistent with a gap: %+v", stats)
	}
	if stats.Issued != 5 || stats.MaxSeq != 6 || stats.Gaps != 1 || stats.DuplicateSeqs != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(stats.MissingSeqs) != 1 || stats.MissingSeqs[0] != 3 {
		t.Fatalf("missing seqs = %v, want [3]", stats.MissingSeqs)
	}

	// 同一序号出现两行 → 重号也能查出来
	dupCode := fmt.Sprintf("DU%08d", batchID)
	if _, err := db.Exec(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, 2)`, batchID, dupCode); err != nil {
		t.Fatalf("insert duplicate seq: %v", err)
	}
	stats, err = svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.Consistent || stats.DuplicateSeqs != 1 || stats.Issued != 6 {
		t.Fatalf("duplicate seq not detected: %+v", stats)
	}
}

// 健康批次连续发码：对账一致
func TestStatsConsistentAfterSequentialIssue(t *testing.T) {
	db := testDB(t)
	batchID := setupBatch(t, db)
	svc := newSvc(db)

	for i := 0; i < 3; i++ {
		r, err := svc.GenerateCodes(batchID, 50)
		if err != nil {
			t.Fatalf("generate round %d: %v", i, err)
		}
		if r.Count != 50 || r.Skipped != 0 {
			t.Fatalf("round %d: count=%d skipped=%d, want 50/0", i, r.Count, r.Skipped)
		}
	}

	stats, err := svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if !stats.Consistent || stats.Issued != 150 || stats.MaxSeq != 150 {
		t.Fatalf("stats not consistent: %+v", stats)
	}

	if _, err := svc.GetCodeStats(999999999); err == nil {
		t.Fatal("expected error for missing batch, got nil")
	}
}
