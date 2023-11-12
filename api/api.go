package api

import (
	"app/api/handler"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	api := r.Group("/api")
	v1 := api.Group("/v1")
	handler.SetupHandler(v1)
}
