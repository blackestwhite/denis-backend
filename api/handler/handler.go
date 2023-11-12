package handler

import (
	"app/api/middleware"
	"app/config"
	"app/entity"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/blackestwhite/presenter"
	"github.com/gin-gonic/gin"
)

func SetupHandler(r *gin.RouterGroup) {
	r.POST("/gen", middleware.Rate(), gen)
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
