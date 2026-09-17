package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
)

type FarmService struct {
	repo *repository.FarmRepo
}

func NewFarmService(repo *repository.FarmRepo) *FarmService {
	return &FarmService{repo: repo}
}

func (s *FarmService) Create(req *model.CreateFarmRequest) (*model.Farm, error) {
	farm := &model.Farm{
		Name:       req.Name,
		RegionCode: req.RegionCode,
		ContactRef: req.ContactRef,
		CertNo:     req.CertNo,
	}
	if err := s.repo.Create(farm); err != nil {
		return nil, err
	}
	return farm, nil
}

func (s *FarmService) GetByID(id int64) (*model.Farm, error) {
	return s.repo.GetByID(id)
}

func (s *FarmService) List() ([]model.Farm, error) {
	return s.repo.List()
}