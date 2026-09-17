package model

import "time"

type TraceCode struct {
	ID              int64      `db:"id" json:"id"`
	BatchID         int64      `db:"batch_id" json:"batch_id"`
	Code            string     `db:"code" json:"code"`
	Seq             int        `db:"seq" json:"seq"`
	PrintedAt       *time.Time `db:"printed_at" json:"printed_at,omitempty"`
	FirstScannedAt  *time.Time `db:"first_scanned_at" json:"first_scanned_at,omitempty"`
	FirstScanRegion *string    `db:"first_scan_region" json:"first_scan_region,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type GenerateCodeRequest struct {
	Count int `json:"count" binding:"required,min=1,max=1000"`
}

type TraceResponse struct {
	Code       string              `json:"code"`
	Batch      *TraceBatchInfo     `json:"batch"`
	Farm       *TraceFarmInfo      `json:"farm"`
	Activities []TraceActivityInfo `json:"activities"`
	Inspection *TraceInspectionInfo `json:"inspection,omitempty"`
	FirstScan  bool                `json:"first_scan"`
}

type TraceBatchInfo struct {
	CropID     string `json:"crop_id"`
	SowingDate string `json:"sowing_date"`
	HarvestDate string `json:"harvest_date,omitempty"`
}

type TraceFarmInfo struct {
	Name       string `json:"name"`
	RegionCode string `json:"region_code"`
	PlotName   string `json:"plot_name"`
}

type TraceActivityInfo struct {
	Kind       ActivityKind `json:"kind"`
	HappenedAt string       `json:"happened_at"`
	Operator   string       `json:"operator"`
	InputName  string       `json:"input_name,omitempty"`
	Dose       *float64     `json:"dose,omitempty"`
	DoseUnit   *string      `json:"dose_unit,omitempty"`
}

type TraceInspectionInfo struct {
	Lab       string           `json:"lab"`
	SampledAt string           `json:"sampled_at"`
	Result    InspectionResult `json:"result"`
}