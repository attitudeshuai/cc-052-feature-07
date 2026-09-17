package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"time"
)

type InspectionService struct {
	repo          *repository.InspectionRepo
	batchRepo     *repository.BatchRepo
}

func NewInspectionService(repo *repository.InspectionRepo, batchRepo *repository.BatchRepo) *InspectionService {
	return &InspectionService{repo: repo, batchRepo: batchRepo}
}

func (s *InspectionService) Create(batchID int64, req *model.CreateInspectionRequest) (*model.Inspection, error) {
	sampledAt, err := time.Parse(time.RFC3339, req.SampledAt)
	if err != nil {
		sampledAt, err = time.Parse("2006-01-02", req.SampledAt)
		if err != nil {
			return nil, err
		}
	}

	items := req.Items
	if items == "" {
		items = "[]"
	}
	insp := &model.Inspection{
		BatchID:   batchID,
		Lab:       req.Lab,
		SampledAt: sampledAt,
		Result:    req.Result,
		ReportURL: req.ReportURL,
		Items:     items,
	}
	if err := s.repo.Create(insp); err != nil {
		return nil, err
	}

	// If inspection failed, lock the batch
	if req.Result == model.InspectionFail {
		s.batchRepo.UpdateStatus(batchID, model.BatchStatusLocked)
	}

	return insp, nil
}

func (s *InspectionService) GetByBatch(batchID int64) (*model.Inspection, error) {
	return s.repo.GetByBatch(batchID)
}