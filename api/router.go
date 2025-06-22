package handler

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	// Initialize router
	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// Initialize news service
	newsService := NewNewsService()

	// Setup routes
	api := r.Group("/api/v1")
	{
		api.GET("/news", newsService.GetAllNews)
		api.GET("/news/:source", newsService.GetNewsBySource)
		api.GET("/sources", newsService.GetAvailableSources)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now()})
		})
	}

	return r
}
