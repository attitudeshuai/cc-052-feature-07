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

// GenerateCodesResult 一次发码的结果：以实际落库为准，撞重跳过的单独列出。
type GenerateCodesResult struct {
	Codes        []string `json:"codes"`                   // 实际落库的码
	Count        int      `json:"count"`                   // 实际发出条数（= 落库行数）
	Skipped      int      `json:"skipped"`                 // 撞唯一约束被跳过的条数
	SkippedCodes []string `json:"skipped_codes,omitempty"` // 被跳过的码
}

// TraceCodeStats 批次发码对账：已分配序号 vs 实际落库，对不上时能查出来。
type TraceCodeStats struct {
	BatchID       int64 `json:"batch_id"`
	Issued        int   `json:"issued"`                 // trace_code 实际行数
	MaxSeq        int   `json:"max_seq"`                // 已分配的最大序号
	Gaps          int   `json:"gaps"`                   // 序号被分配但未落库的个数
	DuplicateSeqs int   `json:"duplicate_seqs"`         // 同一序号重复占用的条数
	Consistent    bool  `json:"consistent"`             // 落库数 == 最大序号 且无重号
	MissingSeqs   []int `json:"missing_seqs,omitempty"` // 缺号明细（最多 100 条）
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