package middleware

import (
	"app/db"
	"fmt"
	"net/http"
	"time"

	"github.com/blackestwhite/presenter"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func Rate() gin.HandlerFunc {
	return func(c *gin.Context) {
		pool := db.RedisPool.Get()
		defer pool.Close()
		now := time.Now().UnixNano()
		key := fmt.Sprint(c.ClientIP(), ":", "rate")

		// Define rate limit parameters.
		maxRequests := 20
		perDuration := time.Hour

		// Calculate the time window.
		windowStart := now - int64(perDuration)

		// Count the number of requests made within the time window.
		count, err := redis.Int(pool.Do("ZCOUNT", key, windowStart, now))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
				Ok:               false,
				ErrorCode:        http.StatusInternalServerError,
				ErrorDescription: err.Error(),
			})
			return
		}

		// Check if the number of requests exceeds the rate limit.
		if count >= maxRequests {
			c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
				Ok:               false,
				ErrorCode:        http.StatusInternalServerError,
				ErrorDescription: "rate limit exceeded",
			})
			return
		}

		// If not exceeded, add the current request's timestamp to the sorted set.
		_, err = pool.Do("ZADD", key, now, now)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
				Ok:               false,
				ErrorCode:        http.StatusInternalServerError,
				ErrorDescription: err.Error(),
			})
			return
		}

		// Expire the key after the rate limiting window to save memory.
		_, err = pool.Do("EXPIRE", key, perDuration.Seconds())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
				Ok:               false,
				ErrorCode:        http.StatusInternalServerError,
				ErrorDescription: err.Error(),
			})
			return
		}

		c.Next()
	}
}
