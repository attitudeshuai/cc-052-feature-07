package repository

import (
	"cc-052/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type BatchRepo struct {
	db *sqlx.DB
}

func NewBatchRepo(db *sqlx.DB) *BatchRepo {
	return &BatchRepo{db: db}
}

func (r *BatchRepo) Create(b *model.CropBatch) error {
	query := `INSERT INTO crop_batch (plot_id, crop_id, sowing_date, harvest_date, expected_yield_kg, status) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	return r.db.QueryRow(query, b.PlotID, b.CropID, b.SowingDate, b.HarvestDate, b.ExpectedYieldKg, b.Status).
		Scan(&b.ID, &b.CreatedAt)
}

func (r *BatchRepo) GetByID(id int64) (*model.CropBatch, error) {
	var b model.CropBatch
	query := `SELECT id, plot_id, crop_id, sowing_date, harvest_date, expected_yield_kg, status, created_at FROM crop_batch WHERE id = $1`
	if err := r.db.Get(&b, query, id); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BatchRepo) UpdateStatus(id int64, status model.BatchStatus) error {
	query := `UPDATE crop_batch SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *BatchRepo) SetHarvestDate(id int64, harvestDate time.Time) error {
	query := `UPDATE crop_batch SET harvest_date = $1, status = 'harvested' WHERE id = $2`
	_, err := r.db.Exec(query, harvestDate, id)
	return err
}

func (r *BatchRepo) GetLastPesticideDate(batchID int64) (*time.Time, error) {
	var t time.Time
	query := `SELECT MAX(a.happened_at) FROM activity a 
	          JOIN input_material im ON a.input_id = im.id 
	          WHERE a.batch_id = $1 AND a.kind = 'pesticide' AND im.safe_interval_days > 0`
	if err := r.db.Get(&t, query, batchID); err != nil {
		return nil, nil // no pesticide found
	}
	return &t, nil
}

func (r *BatchRepo) GetMaxSafeInterval(batchID int64) (int, error) {
	var days int
	query := `SELECT COALESCE(MAX(im.safe_interval_days), 0) FROM activity a 
	          JOIN input_material im ON a.input_id = im.id 
	          WHERE a.batch_id = $1 AND a.kind = 'pesticide'`
	if err := r.db.Get(&days, query, batchID); err != nil {
		return 0, nil
	}
	return days, nil
}