package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"cc-052/pkg/tracecode"
	"fmt"
)

type TraceCodeService struct {
	codeRepo       *repository.TraceCodeRepo
	batchRepo      *repository.BatchRepo
	inspectionRepo *repository.InspectionRepo
	activityRepo   *repository.ActivityRepo
	plotRepo       *repository.PlotRepo
	farmRepo       *repository.FarmRepo
}

func NewTraceCodeService(
	codeRepo *repository.TraceCodeRepo,
	batchRepo *repository.BatchRepo,
	inspectionRepo *repository.InspectionRepo,
	activityRepo *repository.ActivityRepo,
	plotRepo *repository.PlotRepo,
	farmRepo *repository.FarmRepo,
) *TraceCodeService {
	return &TraceCodeService{
		codeRepo:       codeRepo,
		batchRepo:      batchRepo,
		inspectionRepo: inspectionRepo,
		activityRepo:   activityRepo,
		plotRepo:       plotRepo,
		farmRepo:       farmRepo,
	}
}

func (s *TraceCodeService) GenerateCodes(batchID int64, count int) (*model.GenerateCodeResult, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	// Check batch exists
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, fmt.Errorf("batch not found: %w", err)
	}

	// Check if batch is locked
	if batch.Status == model.BatchStatusLocked {
		return nil, fmt.Errorf("batch is locked, cannot generate codes")
	}

	// Check inspection - must have passed
	passed, err := s.inspectionRepo.HasPassedInspection(batchID)
	if err != nil || !passed {
		return nil, fmt.Errorf("batch has not passed inspection")
	}

	// Check safety interval
	ok, msg := s.checkSafetyInterval(batch)
	if !ok {
		return nil, fmt.Errorf("safety interval check failed: %s", msg)
	}

	// Serialize generators of the same batch on the batch row lock: a
	// concurrent request blocks in GetByIDForUpdate until this transaction
	// commits, then reads the new max seq and gets the NEXT segment.
	// One number segment is handed out exactly once.
	tx, err := s.codeRepo.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	lockedBatch, err := s.batchRepo.GetByIDForUpdate(tx, batchID)
	if err != nil {
		return nil, fmt.Errorf("lock batch: %w", err)
	}
	if lockedBatch.Status == model.BatchStatusLocked {
		return nil, fmt.Errorf("batch is locked, cannot generate codes")
	}

	// Quota check while the lock is held: reject requests that would exceed
	// the remaining quota instead of silently issuing a partial batch.
	if lockedBatch.CodeQuota != nil {
		issued, err := s.codeRepo.CountByBatchTx(tx, batchID)
		if err != nil {
			return nil, err
		}
		remaining := *lockedBatch.CodeQuota - issued
		if count > remaining {
			return nil, fmt.Errorf("剩余可发数量不足：剩余 %d，请求 %d", remaining, count)
		}
	}

	maxSeq, err := s.codeRepo.GetMaxSeqByBatchTx(tx, batchID)
	if err != nil {
		return nil, err
	}

	codes := make([]model.TraceCode, 0, count)
	for i := 0; i < count; i++ {
		seq := maxSeq + i + 1
		codes = append(codes, model.TraceCode{
			BatchID: batchID,
			Code:    tracecode.Generate(int64(seq)),
			Seq:     seq,
		})
	}

	inserted, skipped, err := s.codeRepo.BatchInsertTx(tx, codes)
	if err != nil {
		return nil, fmt.Errorf("batch insert codes: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit codes: %w", err)
	}

	// Report what actually landed in the database: Codes/Inserted only
	// contain persisted rows, duplicates skipped on conflict are listed
	// separately in SkippedCodes.
	result := &model.GenerateCodeResult{
		BatchID:   batchID,
		Requested: count,
		Inserted:  len(inserted),
		Skipped:   len(skipped),
		Codes:     make([]string, 0, len(inserted)),
	}
	for _, tc := range inserted {
		result.Codes = append(result.Codes, tc.Code)
	}
	if len(skipped) > 0 {
		result.SkippedCodes = make([]model.SkippedCode, 0, len(skipped))
		for _, tc := range skipped {
			result.SkippedCodes = append(result.SkippedCodes, model.SkippedCode{Seq: tc.Seq, Code: tc.Code})
		}
	}
	return result, nil
}

// GetCodeStats returns the per-batch reconciliation view: how many codes
// are actually persisted, the highest seq consumed, the gap between the
// two (consumed but never persisted) with the missing seq numbers, and
// the remaining quota when one is set.
func (s *TraceCodeService) GetCodeStats(batchID int64) (*model.CodeStats, error) {
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, fmt.Errorf("batch not found: %w", err)
	}

	issued, maxSeq, missing, err := s.codeRepo.GetSeqStats(batchID)
	if err != nil {
		return nil, err
	}

	stats := &model.CodeStats{
		BatchID:     batchID,
		Issued:      issued,
		MaxSeq:      maxSeq,
		Gap:         maxSeq - issued,
		MissingSeqs: missing,
		Quota:       batch.CodeQuota,
	}
	if batch.CodeQuota != nil {
		remaining := *batch.CodeQuota - issued
		stats.Remaining = &remaining
	}
	return stats, nil
}

func (s *TraceCodeService) Trace(code string, region string) (*model.TraceResponse, error) {
	tc, err := s.codeRepo.GetByCode(code)
	if err != nil {
		return nil, fmt.Errorf("code not found: %w", err)
	}

	isFirstScan := tc.FirstScannedAt == nil
	if isFirstScan {
		s.codeRepo.MarkScanned(tc.ID, region)
	}

	batch, err := s.batchRepo.GetByID(tc.BatchID)
	if err != nil {
		return nil, err
	}

	plot, err := s.plotRepo.GetByID(batch.PlotID)
	if err != nil {
		return nil, err
	}

	farm, err := s.farmRepo.GetByID(plot.FarmID)
	if err != nil {
		return nil, err
	}

	activities, err := s.activityRepo.ListByBatch(tc.BatchID)
	if err != nil {
		return nil, err
	}

	inspection, _ := s.inspectionRepo.GetByBatch(tc.BatchID)

	resp := &model.TraceResponse{
		Code:      code,
		FirstScan: isFirstScan,
		Batch: &model.TraceBatchInfo{
			CropID:      batch.CropID,
			SowingDate:  batch.SowingDate.Format("2006-01-02"),
			HarvestDate: "",
		},
		Farm: &model.TraceFarmInfo{
			Name:       farm.Name,
			RegionCode: farm.RegionCode,
			PlotName:   plot.Name,
		},
		Activities: make([]model.TraceActivityInfo, 0),
	}
	if batch.HarvestDate != nil {
		resp.Batch.HarvestDate = batch.HarvestDate.Format("2006-01-02")
	}

	for _, a := range activities {
		info := model.TraceActivityInfo{
			Kind:       a.Kind,
			HappenedAt: a.HappenedAt.Format("2006-01-02"),
			Operator:   a.Operator,
			Dose:       a.Dose,
			DoseUnit:   a.DoseUnit,
		}
		resp.Activities = append(resp.Activities, info)
	}

	if inspection != nil {
		resp.Inspection = &model.TraceInspectionInfo{
			Lab:       inspection.Lab,
			SampledAt: inspection.SampledAt.Format("2006-01-02"),
			Result:    inspection.Result,
		}
	}

	return resp, nil
}

func (s *TraceCodeService) checkSafetyInterval(batch *model.CropBatch) (bool, string) {
	if batch.HarvestDate == nil {
		return true, ""
	}

	lastPesticideDate, err := s.batchRepo.GetLastPesticideDate(batch.ID)
	if err != nil || lastPesticideDate == nil {
		return true, ""
	}

	maxInterval, err := s.batchRepo.GetMaxSafeInterval(batch.ID)
	if err != nil || maxInterval == 0 {
		return true, ""
	}

	daysSincePesticide := int(batch.HarvestDate.Sub(*lastPesticideDate).Hours() / 24)
	if daysSincePesticide < maxInterval {
		return false, fmt.Sprintf("距上次施药%d天，不足安全间隔期%d天", daysSincePesticide, maxInterval)
	}
	return true, ""
}

func (s *TraceCodeService) ValidateTraceCode(code string) bool {
	return tracecode.Validate(code)
}