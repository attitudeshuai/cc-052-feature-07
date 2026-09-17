package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"time"
)

type BatchService struct {
	repo       *repository.BatchRepo
	plotRepo   *repository.PlotRepo
	farmRepo   *repository.FarmRepo
}

func NewBatchService(repo *repository.BatchRepo, plotRepo *repository.PlotRepo, farmRepo *repository.FarmRepo) *BatchService {
	return &BatchService{repo: repo, plotRepo: plotRepo, farmRepo: farmRepo}
}

func (s *BatchService) Create(req *model.CreateBatchRequest) (*model.CropBatch, error) {
	sowingDate, err := time.Parse("2006-01-02", req.SowingDate)
	if err != nil {
		return nil, err
	}
	var harvestDate *time.Time
	if req.HarvestDate != "" {
		t, err := time.Parse("2006-01-02", req.HarvestDate)
		if err != nil {
			return nil, err
		}
		harvestDate = &t
	}
	b := &model.CropBatch{
		PlotID:          req.PlotID,
		CropID:          req.CropID,
		SowingDate:      sowingDate,
		HarvestDate:     harvestDate,
		ExpectedYieldKg: req.ExpectedYieldKg,
		Status:          model.BatchStatusGrowing,
	}
	if err := s.repo.Create(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *BatchService) GetByID(id int64) (*model.CropBatch, error) {
	return s.repo.GetByID(id)
}

func (s *BatchService) CheckSafetyInterval(batchID int64) (bool, string) {
	batch, err := s.repo.GetByID(batchID)
	if err != nil || batch.HarvestDate == nil {
		return true, ""
	}

	lastPesticideDate, err := s.repo.GetLastPesticideDate(batchID)
	if err != nil || lastPesticideDate == nil {
		return true, ""
	}

	maxInterval, err := s.repo.GetMaxSafeInterval(batchID)
	if err != nil || maxInterval == 0 {
		return true, ""
	}

	daysSincePesticide := int(batch.HarvestDate.Sub(*lastPesticideDate).Hours() / 24)
	if daysSincePesticide < maxInterval {
		return false, "距上次施药不足安全间隔期"
	}
	return true, ""
}