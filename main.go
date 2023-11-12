package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/blackestwhite/presenter"
	"github.com/gin-gonic/gin"
)

var key = ""

func init() {
	key = os.Getenv("OPEN_AI_KEY")
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
	router.POST("/api/v1/gen", gen)
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
