package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/seo-optimizer/backend/logging"
)

// StatsMiddleware tracks various statistics about requests
func StatsMiddleware(stats *logging.Statistics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get the real IP address
			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				ip = r.Header.Get("X-Forwarded-For")
			}
			if ip == "" {
				ip = strings.Split(r.RemoteAddr, ":")[0]
			}

			// Track unique visitor
			stats.TrackVisitor(ip)

			// Call the next handler
			next.ServeHTTP(w, r)

			// Only track analysis requests
			if r.URL.Path == "/api/analyze" && r.Method == "POST" {
				loadTime := float64(time.Since(start).Milliseconds())
				stats.TrackAnalysis(r.URL.String(), loadTime, false) // We could add error tracking here
			}

			// Periodically save statistics (every 100 requests or every hour)
			if stats.GetStatistics()["totalRequests"].(int)%100 == 0 {
				go stats.Save() // Save asynchronously to not block the request
			}
		})
	}
} 