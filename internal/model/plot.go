package model

import "time"

type Plot struct {
	ID      int64  `db:"id" json:"id"`
	FarmID  int64  `db:"farm_id" json:"farm_id"`
	Name    string `db:"name" json:"name"`
	AreaMu  float64 `db:"area_mu" json:"area_mu"`
	GeoJSON string `db:"geojson" json:"geojson"`
	SoilType string `db:"soil_type" json:"soil_type"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CreatePlotRequest struct {
	FarmID   int64   `json:"farm_id" binding:"required"`
	Name     string  `json:"name" binding:"required"`
	AreaMu   float64 `json:"area_mu" binding:"required"`
	GeoJSON  string  `json:"geojson"`
	SoilType string  `json:"soil_type"`
}