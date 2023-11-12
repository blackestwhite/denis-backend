package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/blackestwhite/presenter"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var key = ""
var RedisPool *redis.Pool

func init() {
	key = os.Getenv("OPEN_AI_KEY")

	RedisPool = &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", os.Getenv("REDIS_URL"))
			if err != nil {
				log.Printf("ERROR: fail init redis pool: %s", err.Error())
				os.Exit(1)
			}
			return conn, err
		},
	}
}

type ChatCompletion struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRes struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}
type Usage struct {
	PromptTokens    int `json:"prompt_tokens"`
	CompletionToken int `json:"completion_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

func main() {
	router := gin.New()
	router.POST("/api/v1/gen", Rate(), gen)
	router.Run(":8080")
}

type Prompt struct {
	Content []Message `json:"content"`
}

func gen(c *gin.Context) {
	var prompt Prompt
	err := c.Bind(&prompt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
			Ok:               false,
			ErrorCode:        http.StatusInternalServerError,
			ErrorDescription: err.Error(),
		})
		return
	}
	chatCompletion := ChatCompletion{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
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
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", key))
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
	var resp ChatRes
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
		pool := RedisPool.Get()
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
