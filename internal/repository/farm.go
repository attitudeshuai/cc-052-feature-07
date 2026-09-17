package repository

import (
	"cc-052/internal/model"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type FarmRepo struct {
	db *sqlx.DB
}

func NewFarmRepo(db *sqlx.DB) *FarmRepo {
	return &FarmRepo{db: db}
}

func (r *FarmRepo) Create(farm *model.Farm) error {
	query := `INSERT INTO farm (name, region_code, contact_ref, cert_no) 
	          VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRow(query, farm.Name, farm.RegionCode, farm.ContactRef, farm.CertNo).
		Scan(&farm.ID, &farm.CreatedAt)
}

func (r *FarmRepo) GetByID(id int64) (*model.Farm, error) {
	var f model.Farm
	query := `SELECT id, name, region_code, contact_ref, cert_no, created_at FROM farm WHERE id = $1`
	if err := r.db.Get(&f, query, id); err != nil {
		return nil, fmt.Errorf("farm not found: %w", err)
	}
	return &f, nil
}

func (r *FarmRepo) List() ([]model.Farm, error) {
	var farms []model.Farm
	query := `SELECT id, name, region_code, contact_ref, cert_no, created_at FROM farm ORDER BY id`
	if err := r.db.Select(&farms, query); err != nil {
		return nil, err
	}
	return farms, nil
}