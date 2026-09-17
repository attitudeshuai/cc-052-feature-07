package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type BatchHandler struct {
	svc *service.BatchService
}

func NewBatchHandler(svc *service.BatchService) *BatchHandler {
	return &BatchHandler{svc: svc}
}

func (h *BatchHandler) Create(c *gin.Context) {
	var req model.CreateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	batch, err := h.svc.Create(&req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, batch)
}

func (h *BatchHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	batch, err := h.svc.GetByID(id)
	if err != nil {
		response.NotFound(c, "batch not found")
		return
	}
	response.Success(c, batch)
}