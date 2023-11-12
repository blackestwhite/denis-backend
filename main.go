package main

import (
	"app/api"
	"app/config"
	"app/db"

	"github.com/gin-gonic/gin"
)

func init() {
	config.Load()
	db.InitRedis()
}

func main() {
	router := gin.New()

	api.Setup(router)

	router.Run(":8080")
}
