package main

import (
	"log"

	_ "listing/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Listing API
// @version         1.0
// @description     A simple listing service API
// @host            localhost:8080
// @BasePath        /api/v1

func main() {
	r := gin.Default()

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", HealthCheck)
	}

	// Swagger documentation endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

// HealthCheck godoc
// @Summary      Health check endpoint
// @Description  Get the health status of the API
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}
