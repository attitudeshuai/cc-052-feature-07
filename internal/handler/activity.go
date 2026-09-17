package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type ActivityHandler struct {
	svc *service.ActivityService
}

func NewActivityHandler(svc *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

func (h *ActivityHandler) Create(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	var req model.CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	activity, err := h.svc.Create(batchID, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, activity)
}

func (h *ActivityHandler) BatchCreate(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	var req model.BatchCreateActivitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	count, err := h.svc.BatchCreate(batchID, req.Activities)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, gin.H{"inserted": count, "total": len(req.Activities)})
}

func (h *ActivityHandler) ListByBatch(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}
	activities, err := h.svc.ListByBatch(batchID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, activities)
}