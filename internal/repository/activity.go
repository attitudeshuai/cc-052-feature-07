package repository

import (
	"cc-052/internal/model"

	"github.com/jmoiron/sqlx"
)

type ActivityRepo struct {
	db *sqlx.DB
}

func NewActivityRepo(db *sqlx.DB) *ActivityRepo {
	return &ActivityRepo{db: db}
}

func (r *ActivityRepo) Create(a *model.Activity) error {
	query := `INSERT INTO activity (batch_id, client_uuid, kind, happened_at, input_id, dose, dose_unit, operator, photos, geo)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
	          ON CONFLICT (client_uuid) DO NOTHING
	          RETURNING id, created_at`
	return r.db.QueryRow(query, a.BatchID, a.ClientUUID, a.Kind, a.HappenedAt,
		a.InputID, a.Dose, a.DoseUnit, a.Operator, a.Photos, a.Geo).
		Scan(&a.ID, &a.CreatedAt)
}

func (r *ActivityRepo) BatchCreate(activities []model.Activity) (int, error) {
	insertedCount := 0
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`INSERT INTO activity (batch_id, client_uuid, kind, happened_at, input_id, dose, dose_unit, operator, photos, geo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (client_uuid) DO NOTHING`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, a := range activities {
		result, err := stmt.Exec(a.BatchID, a.ClientUUID, a.Kind, a.HappenedAt,
			a.InputID, a.Dose, a.DoseUnit, a.Operator, a.Photos, a.Geo)
		if err != nil {
			return 0, err
		}
		if n, _ := result.RowsAffected(); n > 0 {
			insertedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return insertedCount, nil
}

func (r *ActivityRepo) ListByBatch(batchID int64) ([]model.Activity, error) {
	var activities []model.Activity
	query := `SELECT id, batch_id, client_uuid, kind, happened_at, input_id, dose, dose_unit, operator, photos, geo, created_at 
	          FROM activity WHERE batch_id = $1 ORDER BY happened_at ASC`
	if err := r.db.Select(&activities, query, batchID); err != nil {
		return nil, err
	}
	return activities, nil
}