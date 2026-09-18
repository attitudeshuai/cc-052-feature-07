package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Integration tests for code generation. They need a real PostgreSQL and
// run only when TEST_PG_DSN is set, e.g.:
//
//	TEST_PG_DSN="postgres://farm@127.0.0.1:55432/farm_trace?sslmode=disable" \
//	  go test ./internal/service/ -run TestGenerate -v
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set, skipping integration test")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	if err := repository.RunMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	_, err = db.Exec(`TRUNCATE trace_code, inspection, activity, crop_batch, plot, farm RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// newBatchWithInspection seeds farm/plot/batch (with optional code quota)
// plus a passed inspection, and returns a fully wired TraceCodeService.
func newBatchWithInspection(t *testing.T, db *sqlx.DB, quota *int) (*TraceCodeService, int64) {
	t.Helper()

	var farmID, plotID, batchID int64
	if err := db.QueryRow(`INSERT INTO farm (name, region_code) VALUES ('tf', '000000') RETURNING id`).Scan(&farmID); err != nil {
		t.Fatalf("seed farm: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO plot (farm_id, name, area_mu) VALUES ($1, 'tp', 1) RETURNING id`, farmID).Scan(&plotID); err != nil {
		t.Fatalf("seed plot: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO crop_batch (plot_id, crop_id, sowing_date, expected_yield_kg, status, code_quota)
	                      VALUES ($1, 'crop', '2026-01-01', 100, 'growing', $2) RETURNING id`, plotID, quota).Scan(&batchID); err != nil {
		t.Fatalf("seed batch: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO inspection (batch_id, lab, sampled_at, result) VALUES ($1, 'lab', NOW(), 'pass')`, batchID); err != nil {
		t.Fatalf("seed inspection: %v", err)
	}

	svc := NewTraceCodeService(
		repository.NewTraceCodeRepo(db),
		repository.NewBatchRepo(db),
		repository.NewInspectionRepo(db),
		repository.NewActivityRepo(db),
		repository.NewPlotRepo(db),
		repository.NewFarmRepo(db),
	)
	return svc, batchID
}

// Concurrent generators for the same batch must serialize: every request
// gets a disjoint segment, every returned code is actually persisted, and
// the seq space ends up contiguous with no duplicates and no gaps.
func TestGenerateCodesConcurrent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc, batchID := newBatchWithInspection(t, db, nil)

	const workers = 4
	const perWorker = 100

	results := make([]*model.GenerateCodeResult, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = svc.GenerateCodes(batchID, perWorker)
		}(w)
	}
	close(start)
	wg.Wait()

	seen := make(map[string]bool)
	total := 0
	for i := 0; i < workers; i++ {
		if errs[i] != nil {
			t.Fatalf("worker %d failed: %v", i, errs[i])
		}
		r := results[i]
		if r.Inserted != perWorker || r.Skipped != 0 {
			t.Errorf("worker %d: inserted=%d skipped=%d, want %d/0", i, r.Inserted, r.Skipped, perWorker)
		}
		if len(r.Codes) != r.Inserted {
			t.Errorf("worker %d: len(codes)=%d != inserted=%d", i, len(r.Codes), r.Inserted)
		}
		for _, c := range r.Codes {
			if seen[c] {
				t.Fatalf("code %s handed out twice", c)
			}
			seen[c] = true
			total++
		}
	}
	if total != workers*perWorker {
		t.Fatalf("total issued codes = %d, want %d", total, workers*perWorker)
	}

	stats, err := svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Issued != workers*perWorker || stats.MaxSeq != workers*perWorker || stats.Gap != 0 {
		t.Errorf("stats = %+v, want issued=%d max_seq=%d gap=0", stats, workers*perWorker, workers*perWorker)
	}
	if len(stats.MissingSeqs) != 0 {
		t.Errorf("missing seqs = %v, want none", stats.MissingSeqs)
	}
}

// Requests beyond the remaining quota are rejected with the remaining
// amount; the stats endpoint reflects the authoritative numbers.
func TestGenerateCodesQuota(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	quota := 150
	svc, batchID := newBatchWithInspection(t, db, &quota)

	r, err := svc.GenerateCodes(batchID, 100)
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}
	if r.Inserted != 100 {
		t.Fatalf("inserted = %d, want 100", r.Inserted)
	}

	if _, err := svc.GenerateCodes(batchID, 100); err == nil {
		t.Fatal("expected quota error, got nil")
	} else if !strings.Contains(err.Error(), "剩余 50") {
		t.Fatalf("quota error should mention remaining 50, got: %v", err)
	}

	r, err = svc.GenerateCodes(batchID, 50)
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if r.Inserted != 50 {
		t.Fatalf("inserted = %d, want 50", r.Inserted)
	}

	stats, err := svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Quota == nil || *stats.Quota != 150 {
		t.Errorf("quota = %v, want 150", stats.Quota)
	}
	if stats.Remaining == nil || *stats.Remaining != 0 {
		t.Errorf("remaining = %v, want 0", stats.Remaining)
	}
	if stats.Issued != 150 {
		t.Errorf("issued = %d, want 150", stats.Issued)
	}
}

// BatchInsertTx must report exactly which rows landed and which were
// skipped on conflict.
func TestBatchInsertTxCountsSkipped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	_, batchID := newBatchWithInspection(t, db, nil)
	codeRepo := repository.NewTraceCodeRepo(db)

	tx, err := codeRepo.Begin()
	if err != nil {
		t.Fatal(err)
	}
	inserted, skipped, err := codeRepo.BatchInsertTx(tx, []model.TraceCode{
		{BatchID: batchID, Code: "DUPLICATE01", Seq: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(inserted) != 1 || len(skipped) != 0 {
		t.Fatalf("first insert: inserted=%d skipped=%d, want 1/0", len(inserted), len(skipped))
	}

	tx2, err := codeRepo.Begin()
	if err != nil {
		t.Fatal(err)
	}
	inserted, skipped, err = codeRepo.BatchInsertTx(tx2, []model.TraceCode{
		{BatchID: batchID, Code: "DUPLICATE01", Seq: 2}, // same code -> skipped
		{BatchID: batchID, Code: "FRESHCODE1", Seq: 3},  // new code -> inserted
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(inserted) != 1 || inserted[0].Code != "FRESHCODE1" {
		t.Fatalf("inserted = %+v, want only FRESHCODE1", inserted)
	}
	if len(skipped) != 1 || skipped[0].Code != "DUPLICATE01" {
		t.Fatalf("skipped = %+v, want only DUPLICATE01", skipped)
	}
}

// GetCodeStats must surface consumed-but-never-persisted seq numbers.
func TestGetCodeStatsGap(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc, batchID := newBatchWithInspection(t, db, nil)

	for _, seq := range []int{1, 2, 5} {
		if _, err := db.Exec(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, $3)`,
			batchID, "MANUALSEQ"+string(rune('A'+seq)), seq); err != nil {
			t.Fatalf("seed code: %v", err)
		}
	}

	stats, err := svc.GetCodeStats(batchID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Issued != 3 || stats.MaxSeq != 5 || stats.Gap != 2 {
		t.Fatalf("stats = %+v, want issued=3 max_seq=5 gap=2", stats)
	}
	if len(stats.MissingSeqs) != 2 || stats.MissingSeqs[0] != 3 || stats.MissingSeqs[1] != 4 {
		t.Fatalf("missing seqs = %v, want [3 4]", stats.MissingSeqs)
	}
}
