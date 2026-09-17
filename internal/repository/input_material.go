package repository

import (
	"cc-052/internal/model"

	"github.com/jmoiron/sqlx"
)

type InputMaterialRepo struct {
	db *sqlx.DB
}

func NewInputMaterialRepo(db *sqlx.DB) *InputMaterialRepo {
	return &InputMaterialRepo{db: db}
}

func (r *InputMaterialRepo) GetByID(id int64) (*model.InputMaterial, error) {
	var m model.InputMaterial
	query := `SELECT id, name, type, registration_no, safe_interval_days, active_ingredient, created_at 
	          FROM input_material WHERE id = $1`
	if err := r.db.Get(&m, query, id); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *InputMaterialRepo) List() ([]model.InputMaterial, error) {
	var materials []model.InputMaterial
	query := `SELECT id, name, type, registration_no, safe_interval_days, active_ingredient, created_at 
	          FROM input_material ORDER BY name`
	if err := r.db.Select(&materials, query); err != nil {
		return nil, err
	}
	return materials, nil
}