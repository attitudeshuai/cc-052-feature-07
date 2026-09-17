package handler

import (
	"cc-052/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"context"
)

type HealthHandler struct {
	db  *sqlx.DB
	rdb *redis.Client
}

func NewHealthHandler(db *sqlx.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx := context.Background()
	status := "ok"

	if err := h.db.Ping(); err != nil {
		status = "db_error"
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		status = "cache_error"
	}

	if status != "ok" {
		response.Error(c, 503, status)
		return
	}
	response.Success(c, gin.H{"status": "ok"})
}