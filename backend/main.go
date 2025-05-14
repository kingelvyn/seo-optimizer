package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/seo-optimizer/backend/analyzer"
	"github.com/seo-optimizer/backend/middleware"
)

var (
	seoAnalyzer  *analyzer.Analyzer
	rateLimiter  *middleware.RateLimiter
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize services
	seoAnalyzer = analyzer.New()
	rateLimiter = middleware.NewRateLimiter(2, 5) // 2 requests per second, bucket size of 5

	// Initialize Gin router
	r := gin.Default()

	// Add middlewares
	r.Use(middleware.ErrorHandler())
	r.Use(rateLimiter.RateLimit())

	// Enhanced CORS configuration
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "https://seo-optimizer.elvynprise.xyz")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// API routes
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			log.Printf("Health check request received from: %s\n", c.Request.RemoteAddr)
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		})

		// SEO analysis endpoints
		api.POST("/analyze", analyzeURL)
	}

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Default port
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func analyzeURL(c *gin.Context) {
	log.Printf("Analyze request received from: %s for URL: %s\n", c.Request.RemoteAddr, c.Request.URL.String())
	var request struct {
		URL string `json:"url" binding:"required,url"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid URL provided",
		})
		return
	}

	analysis, err := seoAnalyzer.Analyze(request.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to analyze URL: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
} 