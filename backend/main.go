package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/seo-optimizer/backend/analyzer"
	"github.com/seo-optimizer/backend/middleware"
)

var (
	seoAnalyzer *analyzer.Analyzer
	rateLimiter *middleware.RateLimiter
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

func setupTrustedProxies(r *gin.Engine) error {
	// In development, trust only localhost
	if os.Getenv("GIN_MODE") == "debug" {
		return r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	// In production, trust Docker's internal network
	dockerNetwork := os.Getenv("DOCKER_NETWORK")
	if dockerNetwork == "" {
		dockerNetwork = "172.0.0.0/8" // Default Docker network
	}

	return r.SetTrustedProxies([]string{dockerNetwork})
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';")

		// Remove sensitive headers
		c.Header("Server", "")
		c.Next()
	}
}

func getRateLimitConfig() (int, int) {
	requestsStr := os.Getenv("RATE_LIMIT_REQUESTS")
	durationStr := os.Getenv("RATE_LIMIT_DURATION")

	requests, err := strconv.Atoi(requestsStr)
	if err != nil || requests <= 0 {
		requests = 2 // Default: 2 requests
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		duration = 1 // Default: 1 second
	}

	return requests, duration
}

func initializeAnalyzer() (*analyzer.Analyzer, error) {
	// Get data directory from environment variable
	dataDir := os.Getenv("DATA_DIR")

	// If not set, use different defaults for development and production
	if dataDir == "" {
		if os.Getenv("GIN_MODE") == "release" {
			dataDir = "/app/data" // Docker volume path for production
		} else {
			// For local development, use a directory in the project
			dataDir = "data"
		}
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to initialize stats storage: %w", err)
	}

	// Log the data directory being used
	log.Printf("Using data directory: %s", dataDir)

	// Create analyzer instance
	analyzerInstance, err := analyzer.New(dataDir)
	if err != nil {
		return nil, err
	}

	// Start periodic cleanup in background
	go func() {
		// Calculate duration until next midnight
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		duration := nextMidnight.Sub(now)

		// Wait until first midnight
		time.Sleep(duration)

		// Then run daily at midnight
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		cleanup := func() {
			if stats := analyzerInstance.GetStats(); stats != nil {
				// Keep only current month and previous month
				stats.Cleanup(1) // 1 means keep current month plus 1 previous month
				log.Printf("Statistics cleanup completed at %v", time.Now().Format("2006-01-02 15:04:05"))
			}
		}

		// Run cleanup immediately after midnight
		cleanup()

		// Then run every 24 hours
		for range ticker.C {
			cleanup()
		}
	}()

	return analyzerInstance, nil
}

func main() {
	// Load environment configuration
	loadEnv()

	// Set up Gin mode
	setupGinMode()

	// Initialize services
	var err error
	seoAnalyzer, err = initializeAnalyzer()
	if err != nil {
		log.Fatalf("Failed to initialize analyzer: %v", err)
	}

	requests, duration := getRateLimitConfig()
	rateLimiter = middleware.NewRateLimiter(float64(requests), float64(duration*5)) // Convert to float64

	// Initialize Gin router
	r := gin.Default()

	// Set up trusted proxies
	if err := setupTrustedProxies(r); err != nil {
		log.Printf("Warning: Failed to set trusted proxies: %v\n", err)
	}

	// Add security headers
	r.Use(securityHeaders())

	// Add middlewares
	r.Use(middleware.ErrorHandler())
	r.Use(rateLimiter.RateLimit())

	// CORS middleware with more restrictive settings
	r.Use(func(c *gin.Context) {
		// In development, allow all origins
		origin := "*"
		if os.Getenv("GIN_MODE") == "release" {
			// In production, restrict to your domain
			origin = "https://seo-optimizer.elvynprise.xyz"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Convert standard middleware to Gin middleware
	r.Use(func(c *gin.Context) {
		// Get the real IP address
		ip := c.ClientIP()

		// Track unique visitor using the new stats system
		if stats := seoAnalyzer.GetStats(); stats != nil {
			stats.TrackVisitor(ip)
		}

		c.Next()
	})

	// API routes
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			log.Printf("Health check request received from: %s\n", c.ClientIP())
			log.Printf("Health check headers: %v\n", c.Request.Header)

			// Get cache statistics
			cacheStats := seoAnalyzer.GetCacheStats()
			log.Printf("Cache stats: %+v\n", cacheStats)

			// Get current stats
			currentStats := seoAnalyzer.GetStats().GetCurrentStats()
			log.Printf("Current stats: %+v\n", currentStats)

			// Calculate memory stats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("Memory stats: Alloc=%v, TotalAlloc=%v, Sys=%v, NumGC=%v\n",
				m.Alloc, m.TotalAlloc, m.Sys, m.NumGC)

			// Log goroutine count
			log.Printf("[DEBUG] Goroutine count: %d\n", runtime.NumGoroutine())

			// Prepare health response
			health := gin.H{
				"status":    "ok",
				"timestamp": time.Now().Format(time.RFC3339),
				"cache": gin.H{
					"analysisEntries":     cacheStats.AnalysisEntries,
					"linkEntries":         cacheStats.LinkEntries,
					"analysisCacheHits":   cacheStats.AnalysisCacheHits,
					"linkCacheHits":       cacheStats.LinkCacheHits,
					"analysisCacheMisses": cacheStats.AnalysisCacheMisses,
					"linkCacheMisses":     cacheStats.LinkCacheMisses,
				},
				"memory": gin.H{
					"alloc":      m.Alloc,
					"totalAlloc": m.TotalAlloc,
					"sys":        m.Sys,
					"numGC":      m.NumGC,
				},
				"stats": gin.H{
					"errorRate":         currentStats.ErrorCount,
					"totalRequests":     currentStats.TotalRequests,
					"uniqueVisitors24h": len(currentStats.UniqueVisitors),
				},
			}

			log.Printf("Sending health response: %+v\n", health)
			c.JSON(http.StatusOK, health)
		})

		// SEO analysis endpoints
		api.POST("/analyze", analyzeURL)

		// Cache status endpoint
		api.GET("/cache-status", getCacheStatus)

		// Statistics endpoint
		api.GET("/statistics", func(c *gin.Context) {
			if stats := seoAnalyzer.GetStats(); stats != nil {
				currentStats := stats.GetCurrentStats()

				// Filter out /api/analyze from popularUrls and adjust counters
				filteredUrls := make(map[string]int)
				apiCallCount := 0
				if currentStats.PopularUrls != nil {
					for url, count := range currentStats.PopularUrls {
						if url != "/api/analyze" {
							filteredUrls[url] = count
						} else {
							apiCallCount = count
						}
					}
				}

				// Adjust total requests to exclude API calls
				adjustedRequests := currentStats.TotalRequests - apiCallCount
				if adjustedRequests < 0 {
					adjustedRequests = 0
				}

				// Calculate average load time based on actual analyses
				var avgLoadTime float64
				if adjustedRequests > 0 {
					avgLoadTime = currentStats.TotalLoadTime / float64(adjustedRequests)
				}

				// Prepare response with all numerical stats
				response := gin.H{
					"uniqueVisitors24h": len(currentStats.UniqueVisitors),
					"totalRequests":     adjustedRequests,
					"errorRate":         float64(currentStats.ErrorCount) / float64(adjustedRequests+1) * 100,
					"averageLoadTime":   avgLoadTime,
				}

				// Include popular URLs only in development mode
				if os.Getenv("GIN_MODE") != "release" {
					response["popularUrls"] = filteredUrls
				}

				c.JSON(http.StatusOK, response)
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Statistics not available"})
			}
		})
	}

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Default port
	}

	// Create a server with graceful shutdown timeout
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Shutdown the analyzer (which will save stats)
	if err := seoAnalyzer.Shutdown(); err != nil {
		log.Printf("Error during analyzer shutdown: %v", err)
	}

	log.Println("Server exited")
}

func analyzeURL(c *gin.Context) {
	start := time.Now()
	log.Printf("Analyze request received from: %s\n", c.ClientIP())
	var request struct {
		URL   string `json:"url" binding:"required,url"`
		Track bool   `json:"track"`
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

	// Track the actual analyzed URL, not the API endpoint
	loadTime := float64(time.Since(start).Milliseconds())
	if stats := seoAnalyzer.GetStats(); stats != nil {
		// Only track if it's a valid URL
		if request.URL != "" && request.URL != "/api/analyze" {
			stats.TrackAnalysis(request.URL, loadTime, false)
			log.Printf("Tracked analysis for URL: %s", request.URL)
		}
	}

	c.JSON(http.StatusOK, analysis)
}

func getCacheStatus(c *gin.Context) {
	log.Printf("Cache status request received from: %s\n", c.ClientIP())

	// Get cache statistics
	stats := seoAnalyzer.GetCacheStats()

	// Check if a specific URL is cached
	url := c.Query("url")
	isCached := false
	if url != "" {
		isCached = seoAnalyzer.IsCached(url)
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":    stats,
		"url":      url,
		"isCached": isCached,
	})
}
