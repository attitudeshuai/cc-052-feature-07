package repository

import (
	"cc-052/internal/model"

	"github.com/jmoiron/sqlx"
)

type PlotRepo struct {
	db *sqlx.DB
}

func NewPlotRepo(db *sqlx.DB) *PlotRepo {
	return &PlotRepo{db: db}
}

func (r *PlotRepo) Create(p *model.Plot) error {
	query := `INSERT INTO plot (farm_id, name, area_mu, geojson, soil_type) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return r.db.QueryRow(query, p.FarmID, p.Name, p.AreaMu, p.GeoJSON, p.SoilType).
		Scan(&p.ID, &p.CreatedAt)
}

func (r *PlotRepo) GetByID(id int64) (*model.Plot, error) {
	var p model.Plot
	query := `SELECT id, farm_id, name, area_mu, geojson, soil_type, created_at FROM plot WHERE id = $1`
	if err := r.db.Get(&p, query, id); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlotRepo) ListByFarm(farmID int64) ([]model.Plot, error) {
	var plots []model.Plot
	query := `SELECT id, farm_id, name, area_mu, geojson, soil_type, created_at FROM plot WHERE farm_id = $1 ORDER BY id`
	if err := r.db.Select(&plots, query, farmID); err != nil {
		return nil, err
	}
	return plots, nil
}