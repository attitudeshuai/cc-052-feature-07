package repository

import (
	"cc-052/internal/model"

	"github.com/jmoiron/sqlx"
)

type InspectionRepo struct {
	db *sqlx.DB
}

func NewInspectionRepo(db *sqlx.DB) *InspectionRepo {
	return &InspectionRepo{db: db}
}

func (r *InspectionRepo) Create(insp *model.Inspection) error {
	query := `INSERT INTO inspection (batch_id, lab, sampled_at, result, report_url, items) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	return r.db.QueryRow(query, insp.BatchID, insp.Lab, insp.SampledAt, insp.Result, insp.ReportURL, insp.Items).
		Scan(&insp.ID, &insp.CreatedAt)
}

func (r *InspectionRepo) GetByBatch(batchID int64) (*model.Inspection, error) {
	var insp model.Inspection
	query := `SELECT id, batch_id, lab, sampled_at, result, report_url, items, created_at 
	          FROM inspection WHERE batch_id = $1 ORDER BY id DESC LIMIT 1`
	if err := r.db.Get(&insp, query, batchID); err != nil {
		return nil, err
	}
	return &insp, nil
}

func (r *InspectionRepo) HasPassedInspection(batchID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM inspection WHERE batch_id = $1 AND result = 'pass'`
	if err := r.db.Get(&count, query, batchID); err != nil {
		return false, err
	}
	return count > 0, nil
}