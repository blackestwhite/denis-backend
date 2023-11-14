package handler

import (
	"app/api/middleware"
	"app/config"
	"app/entity"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/blackestwhite/presenter"
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

var Chans map[string]chan (string)

func SetupHandler(r *gin.RouterGroup) {
	Chans = make(map[string]chan string)
	r.POST("/gen", middleware.Rate(), gen)
	r.GET("/get/:id", get)
}

func get(c *gin.Context) {
	id := c.Param("id")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	ch := Chans[id]
	for {
		// Retrieve data from the channel and write it to the response
		data := <-ch
		// Flush the response to ensure data is sent immediately
		c.Writer.Write([]byte(data + "\n"))
		c.Writer.Flush()

		if strings.Contains(data, "[DONE]") {
			break
		}
	}
	close(ch)
	delete(Chans, id)
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
				Content: `you are a coding assistant built by Mahdi Akbari in [https://akbari.foundation](akbari.foundation).
				your name is Denis which is named after Denis Ritchie.
				act as a senior L3 engineer in google, answer the questions or refactor the codes(if provided). and DO NOT break the working functionality(if code is provided and working).
				answers should be in markdown format.
				`,
			},
		},
		Stream: true,
	}
	chatCompletion.Messages = append(chatCompletion.Messages, prompt.Content...)
	marshalled, err := json.Marshal(chatCompletion)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
			Ok:               false,
			ErrorCode:        http.StatusInternalServerError,
			ErrorDescription: err.Error(),
		})
		return
	}
	wg := &sync.WaitGroup{}
	id := uuid.New()
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		client := &http.Client{}
		req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(marshalled))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
				Ok:               false,
				ErrorCode:        http.StatusInternalServerError,
				ErrorDescription: err.Error(),
			})
			return
		}
		req.Header.Add("Authorization", fmt.Sprint("Bearer ", config.KEY))
		req.Header.Add("Content-Type", "application/json")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Connection", "keep-alive")
		res, err := client.Do(req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, presenter.Std{
				Ok:               false,
				ErrorCode:        http.StatusInternalServerError,
				ErrorDescription: err.Error(),
			})
			return
		}
		Chans[id.String()] = make(chan string, 1)
		wg.Done()
		for {
			// var resp entity.ChatRes
			data := make([]byte, 1024)
			_, err := res.Body.Read(data)
			if err == nil {
				Chans[id.String()] <- string(data)
				if strings.Contains(string(data), "\"finish_reason\":\"stop\"}") {
					break
				}
			}
		}
	}(wg)

	wg.Wait()
	c.JSON(http.StatusOK, presenter.Std{
		Ok: true,
		Result: gin.H{
			"id": id.String(),
		},
	})
}
