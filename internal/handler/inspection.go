package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type InspectionHandler struct {
	svc *service.InspectionService
}

func NewInspectionHandler(svc *service.InspectionService) *InspectionHandler {
	return &InspectionHandler{svc: svc}
}

func (h *InspectionHandler) Create(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	var req model.CreateInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	insp, err := h.svc.Create(batchID, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, insp)
}