package main

import (
	"app/api"
	"app/config"
	"app/db"
	"log"

	"github.com/gin-gonic/gin"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	config.Load()
	db.InitRedis()
}

func main() {
	router := gin.New()

	api.Setup(router)

	router.Run(":8080")
}
