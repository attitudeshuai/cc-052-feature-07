package model

import "time"

type BatchStatus string

const (
	BatchStatusGrowing   BatchStatus = "growing"
	BatchStatusHarvested BatchStatus = "harvested"
	BatchStatusLocked    BatchStatus = "locked"
)

type CropBatch struct {
	ID              int64       `db:"id" json:"id"`
	PlotID          int64       `db:"plot_id" json:"plot_id"`
	CropID          string      `db:"crop_id" json:"crop_id"`
	SowingDate      time.Time   `db:"sowing_date" json:"sowing_date"`
	HarvestDate     *time.Time  `db:"harvest_date" json:"harvest_date,omitempty"`
	ExpectedYieldKg float64     `db:"expected_yield_kg" json:"expected_yield_kg"`
	Status          BatchStatus `db:"status" json:"status"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
}

type CreateBatchRequest struct {
	PlotID          int64   `json:"plot_id" binding:"required"`
	CropID          string  `json:"crop_id" binding:"required"`
	SowingDate      string  `json:"sowing_date" binding:"required"`
	HarvestDate     string  `json:"harvest_date"`
	ExpectedYieldKg float64 `json:"expected_yield_kg" binding:"required"`
}