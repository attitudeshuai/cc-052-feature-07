package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
)

type PlotService struct {
	repo *repository.PlotRepo
}

func NewPlotService(repo *repository.PlotRepo) *PlotService {
	return &PlotService{repo: repo}
}

func (s *PlotService) Create(req *model.CreatePlotRequest) (*model.Plot, error) {
	p := &model.Plot{
		FarmID:   req.FarmID,
		Name:     req.Name,
		AreaMu:   req.AreaMu,
		GeoJSON:  req.GeoJSON,
		SoilType: req.SoilType,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PlotService) GetByID(id int64) (*model.Plot, error) {
	return s.repo.GetByID(id)
}

func (s *PlotService) ListByFarm(farmID int64) ([]model.Plot, error) {
	return s.repo.ListByFarm(farmID)
}