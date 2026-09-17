package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type FarmHandler struct {
	svc *service.FarmService
}

func NewFarmHandler(svc *service.FarmService) *FarmHandler {
	return &FarmHandler{svc: svc}
}

func (h *FarmHandler) Create(c *gin.Context) {
	var req model.CreateFarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	farm, err := h.svc.Create(&req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, farm)
}

func (h *FarmHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	farm, err := h.svc.GetByID(id)
	if err != nil {
		response.NotFound(c, "farm not found")
		return
	}
	response.Success(c, farm)
}

func (h *FarmHandler) List(c *gin.Context) {
	farms, err := h.svc.List()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, farms)
}