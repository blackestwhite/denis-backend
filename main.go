package main

import (
	"app/api"
	"app/config"

	"github.com/gin-gonic/gin"
)

func init() {
	config.Load()
}

func main() {
	router := gin.New()

	api.Setup(router)

	router.Run(":8080")
}
