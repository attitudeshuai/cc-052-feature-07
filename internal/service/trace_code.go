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

func (s *TraceCodeService) GenerateCodes(batchID int64, count int) ([]string, error) {
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

	// Get max seq
	maxSeq, err := s.codeRepo.GetMaxSeqByBatch(batchID)
	if err != nil {
		return nil, err
	}

	// Generate codes in batch of 1000
	var allCodes []string
	batchSize := 1000
	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}
		size := end - i

		var codes []model.TraceCode
		var codeStrings []string
		for j := 0; j < size; j++ {
			seq := int64(maxSeq + i + j + 1)
			code := tracecode.Generate(seq)
			codes = append(codes, model.TraceCode{
				BatchID: batchID,
				Code:    code,
				Seq:     int(seq),
			})
			codeStrings = append(codeStrings, code)
		}

		if err := s.codeRepo.BatchInsert(codes); err != nil {
			return nil, fmt.Errorf("batch insert codes: %w", err)
		}
		allCodes = append(allCodes, codeStrings...)
	}

	return allCodes, nil
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