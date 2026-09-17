package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"time"
)

type ActivityService struct {
	repo      *repository.ActivityRepo
	batchRepo *repository.BatchRepo
}

func NewActivityService(repo *repository.ActivityRepo, batchRepo *repository.BatchRepo) *ActivityService {
	return &ActivityService{repo: repo, batchRepo: batchRepo}
}

func (s *ActivityService) Create(batchID int64, req *model.CreateActivityRequest) (*model.Activity, error) {
	happenedAt, err := time.Parse(time.RFC3339, req.HappenedAt)
	if err != nil {
		// Try date only
		happenedAt, err = time.Parse("2006-01-02", req.HappenedAt)
		if err != nil {
			return nil, err
		}
	}

	// Check batch exists
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, err
	}

	// Time integrity: happened_at not before sowing, not after harvest
	if happenedAt.Before(batch.SowingDate) {
		return nil, nil // silently reject / needs_review
	}
	if batch.HarvestDate != nil && happenedAt.After(*batch.HarvestDate) {
		return nil, nil
	}

	a := &model.Activity{
		BatchID:    batchID,
		ClientUUID: req.ClientUUID,
		Kind:       req.Kind,
		HappenedAt: happenedAt,
		InputID:    req.InputID,
		Dose:       req.Dose,
		DoseUnit:   req.DoseUnit,
		Operator:   req.Operator,
		Photos:     req.Photos,
		Geo:        req.Geo,
	}

	if err := s.repo.Create(a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ActivityService) BatchCreate(batchID int64, reqs []model.CreateActivityRequest) (int, error) {
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return 0, err
	}

	var activities []model.Activity
	for _, r := range reqs {
		happenedAt, err := time.Parse(time.RFC3339, r.HappenedAt)
		if err != nil {
			happenedAt, err = time.Parse("2006-01-02", r.HappenedAt)
			if err != nil {
				continue
			}
		}
		// Skip if time integrity check fails
		if happenedAt.Before(batch.SowingDate) {
			continue
		}
		if batch.HarvestDate != nil && happenedAt.After(*batch.HarvestDate) {
			continue
		}

		activities = append(activities, model.Activity{
			BatchID:    batchID,
			ClientUUID: r.ClientUUID,
			Kind:       r.Kind,
			HappenedAt: happenedAt,
			InputID:    r.InputID,
			Dose:       r.Dose,
			DoseUnit:   r.DoseUnit,
			Operator:   r.Operator,
			Photos:     r.Photos,
			Geo:        r.Geo,
		})
	}

	if len(activities) == 0 {
		return 0, nil
	}

	return s.repo.BatchCreate(activities)
}

func (s *ActivityService) ListByBatch(batchID int64) ([]model.Activity, error) {
	return s.repo.ListByBatch(batchID)
}