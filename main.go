package main

import (
	"app/config"
	"app/db"
	"app/entity"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/blackestwhite/presenter"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func init() {
	config.Load()
}

func main() {
	router := gin.New()
	router.POST("/api/v1/gen", Rate(), gen)
	router.Run(":8080")
}

func gen(c *gin.Context) {
	var prompt entity.Prompt
	err := c.Bind(&prompt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
			Ok:               false,
			ErrorCode:        http.StatusInternalServerError,
			ErrorDescription: err.Error(),
		})
		return
	}
	chatCompletion := entity.ChatCompletion{
		Model: "gpt-3.5-turbo",
		Messages: []entity.Message{
			{
				Role: "system",
				Content: `you are a coding assistant built by Mahdi Akbari and backed by [https://akbari.foundation](akbari.foundation).
				your name is Denis which is named after Denis Ritchie.
				act as a senior L3 engineer in google, answer the questions or refactor the codes(if provided). and DO NOT break the working functionality(if code is provided and working).
				answers should be in markdown format.
				`,
			},
		},
	}
	chatCompletion.Messages = append(chatCompletion.Messages, prompt.Content...)
	marshalled, err := json.Marshal(chatCompletion)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(marshalled))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", config.KEY))
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var resp entity.ChatRes
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Fatal(err)
	}

	c.JSON(http.StatusOK, presenter.Std{
		Ok:     true,
		Result: resp,
	})
}

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
