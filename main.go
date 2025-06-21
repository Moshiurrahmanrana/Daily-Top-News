package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
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
	setupRoutes(r, newsService)

	// Start server
	log.Println("Starting News API server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRoutes(r *gin.Engine, newsService *NewsService) {
	api := r.Group("/api/v1")
	{
		api.GET("/news", newsService.GetAllNews)
		api.GET("/news/:source", newsService.GetNewsBySource)
		api.GET("/sources", newsService.GetAvailableSources)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now()})
		})
	}
}
