package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type PlotHandler struct {
	svc *service.PlotService
}

func NewPlotHandler(svc *service.PlotService) *PlotHandler {
	return &PlotHandler{svc: svc}
}

func (h *PlotHandler) Create(c *gin.Context) {
	var req model.CreatePlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	plot, err := h.svc.Create(&req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, plot)
}

func (h *PlotHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	plot, err := h.svc.GetByID(id)
	if err != nil {
		response.NotFound(c, "plot not found")
		return
	}
	response.Success(c, plot)
}

func (h *PlotHandler) ListByFarm(c *gin.Context) {
	farmID, err := strconv.ParseInt(c.Query("farm_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid farm_id")
		return
	}
	plots, err := h.svc.ListByFarm(farmID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, plots)
}