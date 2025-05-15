package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/seo-optimizer/backend/analyzer"
	"github.com/seo-optimizer/backend/middleware"
	"github.com/seo-optimizer/backend/logging"
)

var (
	seoAnalyzer  *analyzer.Analyzer
	rateLimiter  *middleware.RateLimiter
)

func loadEnv() {
	// Try to load .env.development first (for local development)
	if err := godotenv.Load(".env.development"); err != nil {
		// If .env.development doesn't exist, try regular .env
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}
}

func setupGinMode() {
	// Set Gin mode based on environment variable
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		// Default to release mode if not specified
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)
}

func main() {
	// Load environment configuration
	loadEnv()
	
	// Set up Gin mode
	setupGinMode()

	// Initialize services
	seoAnalyzer = analyzer.New()
	rateLimiter = middleware.NewRateLimiter(2, 5) // 2 requests per second, bucket size of 5

	// Initialize statistics
	stats := logging.Initialize()

	// Initialize Gin router
	r := gin.Default()

	// Add middlewares
	r.Use(middleware.ErrorHandler())
	r.Use(rateLimiter.RateLimit())
	
	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Convert standard middleware to Gin middleware
	r.Use(func(c *gin.Context) {
		start := time.Now()
		
		// Get the real IP address
		ip := c.ClientIP()
		
		// Track unique visitor
		stats.TrackVisitor(ip)
		
		c.Next()
		
		// Only track analysis requests
		if c.Request.URL.Path == "/api/analyze" && c.Request.Method == "POST" {
			loadTime := float64(time.Since(start).Milliseconds())
			stats.TrackAnalysis(c.Request.URL.String(), loadTime, c.Writer.Status() >= 400)
		}
		
		// Periodically save statistics
		if stats.GetStatistics()["totalRequests"].(int)%100 == 0 {
			go stats.Save()
		}
	})

	// API routes
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			log.Printf("Health check request received from: %s\n", c.ClientIP())
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		})

		// SEO analysis endpoints
		api.POST("/analyze", analyzeURL)
		
		// Statistics endpoint
		api.GET("/statistics", func(c *gin.Context) {
			c.JSON(http.StatusOK, stats.GetStatistics())
		})
	}

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Default port
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func analyzeURL(c *gin.Context) {
	log.Printf("Analyze request received from: %s\n", c.ClientIP())
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