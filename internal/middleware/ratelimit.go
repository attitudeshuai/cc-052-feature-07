package middleware

import (
	"cc-052/pkg/response"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func RateLimit(rdb *redis.Client, rate int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		key := "ratelimit:" + c.ClientIP()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(rate) {
			response.Error(c, 429, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}