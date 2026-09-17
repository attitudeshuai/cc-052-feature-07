package model

import "time"

type InspectionResult string

const (
	InspectionPass InspectionResult = "pass"
	InspectionFail InspectionResult = "fail"
)

type Inspection struct {
	ID        int64            `db:"id" json:"id"`
	BatchID   int64            `db:"batch_id" json:"batch_id"`
	Lab       string           `db:"lab" json:"lab"`
	SampledAt time.Time        `db:"sampled_at" json:"sampled_at"`
	Result    InspectionResult `db:"result" json:"result"`
	ReportURL string           `db:"report_url" json:"report_url"`
	Items     string           `db:"items" json:"items"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
}

type CreateInspectionRequest struct {
	Lab       string           `json:"lab" binding:"required"`
	SampledAt string           `json:"sampled_at" binding:"required"`
	Result    InspectionResult `json:"result" binding:"required"`
	ReportURL string           `json:"report_url"`
	Items     string           `json:"items"`
}