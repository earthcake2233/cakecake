package handler

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterSwaggerRoutes mounts the Swagger UI and JSON endpoints in development mode only.
func RegisterSwaggerRoutes(r *gin.Engine, appEnv string) {
	if appEnv != "development" {
		return
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
