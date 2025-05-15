package logging

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Environment variable name for controlling statistics visibility
const ENV_DEV_MODE = "DEV_MODE"

// Statistics represents the collected statistics
type Statistics struct {
	UniqueVisitors     map[string]time.Time `json:"uniqueVisitors"`     // IP -> Last Visit Time
	AnalysisRequests   int                  `json:"analysisRequests"`   // Total number of analysis requests
	ErrorCount         int                  `json:"errorCount"`         // Number of errors
	PopularURLs        map[string]int       `json:"popularUrls"`        // URL -> Count
	AverageLoadTime    float64              `json:"averageLoadTime"`    // Average load time in milliseconds
	TotalLoadTime      float64              `json:"-"`                  // Used to calculate average
	RequestCount       int                  `json:"-"`                  // Used to calculate average
	LastPersisted      time.Time            `json:"lastPersisted"`      // Last time stats were saved
	mutex             sync.RWMutex          `json:"-"`
}

var (
	stats *Statistics
	once sync.Once
)

// Initialize creates or loads the statistics
func Initialize() *Statistics {
	once.Do(func() {
		stats = &Statistics{
			UniqueVisitors:   make(map[string]time.Time),
			PopularURLs:      make(map[string]int),
			LastPersisted:    time.Now(),
		}
		
		// Try to load existing statistics
		if err := stats.Load(); err != nil {
			fmt.Printf("Could not load existing statistics: %v\n", err)
		}
	})
	return stats
}

// TrackVisitor records a unique visitor
func (s *Statistics) TrackVisitor(ip string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.UniqueVisitors[ip] = time.Now()
}

// cleanURL removes API paths and query parameters, returns just the main URL
func cleanURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// Don't track our own API URLs
	if strings.Contains(u.Host, "localhost") || 
	   strings.Contains(u.Host, "127.0.0.1") ||
	   strings.Contains(strings.ToLower(u.Path), "/api/") {
		return ""
	}

	// Build clean URL with just scheme and host
	cleanURL := u.Scheme + "://" + u.Host
	
	// Add path if it exists and isn't just "/"
	if u.Path != "" && u.Path != "/" {
		cleanURL += u.Path
	}

	// Trim trailing slash
	return strings.TrimSuffix(cleanURL, "/")
}

// TrackAnalysis records an analysis request
func (s *Statistics) TrackAnalysis(url string, loadTime float64, hasError bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.AnalysisRequests++
	
	// Clean the URL before storing
	cleanedURL := cleanURL(url)
	// Only track non-empty URLs (those that passed our filtering)
	if cleanedURL != "" {
		s.PopularURLs[cleanedURL]++
	}
	
	if hasError {
		s.ErrorCount++
	}
	
	// Update average load time
	s.TotalLoadTime += loadTime
	s.RequestCount++
	s.AverageLoadTime = s.TotalLoadTime / float64(s.RequestCount)
}

// GetUniqueVisitorsCount returns the number of unique visitors in the last 24 hours
func (s *Statistics) GetUniqueVisitorsCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	count := 0
	cutoff := time.Now().Add(-24 * time.Hour)
	
	for _, lastVisit := range s.UniqueVisitors {
		if lastVisit.After(cutoff) {
			count++
		}
	}
	
	return count
}

// GetPopularURLs returns the top N most analyzed URLs
func (s *Statistics) GetPopularURLs(n int) map[string]int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	result := make(map[string]int)
	count := 0
	
	// Simple implementation - for production, use a heap or sorted data structure
	for url, freq := range s.PopularURLs {
		if count < n {
			result[url] = freq
			count++
		}
	}
	
	return result
}

// GetErrorRate returns the error rate as a percentage
func (s *Statistics) GetErrorRate() float64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if s.AnalysisRequests == 0 {
		return 0
	}
	
	return (float64(s.ErrorCount) / float64(s.AnalysisRequests)) * 100
}

// Save persists the statistics to a file
func (s *Statistics) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.LastPersisted = time.Now()
	
	file, err := os.Create("statistics.json")
	if err != nil {
		return fmt.Errorf("could not create statistics file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(s); err != nil {
		return fmt.Errorf("could not encode statistics: %v", err)
	}
	
	return nil
}

// Load reads the statistics from a file
func (s *Statistics) Load() error {
	file, err := os.Open("statistics.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Not an error if file doesn't exist yet
		}
		return fmt.Errorf("could not open statistics file: %v", err)
	}
	defer file.Close()
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(s); err != nil {
		return fmt.Errorf("could not decode statistics: %v", err)
	}
	
	return nil
}

// GetStatistics returns a copy of the current statistics, but only in development mode
func (s *Statistics) GetStatistics() map[string]interface{} {
	// Check if we're in development mode
	if os.Getenv(ENV_DEV_MODE) != "true" {
		// In production, return limited statistics without sensitive data
		s.mutex.RLock()
		defer s.mutex.RUnlock()
		
		return map[string]interface{}{
			"uniqueVisitors24h": s.GetUniqueVisitorsCount(),
			"totalRequests":     s.AnalysisRequests,
			"errorRate":         s.GetErrorRate(),
			"averageLoadTime":   s.AverageLoadTime,
		}
	}

	// In development mode, return full statistics
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return map[string]interface{}{
		"uniqueVisitors24h": s.GetUniqueVisitorsCount(),
		"totalRequests":     s.AnalysisRequests,
		"errorRate":         s.GetErrorRate(),
		"averageLoadTime":   s.AverageLoadTime,
		"popularUrls":       s.GetPopularURLs(5), // Top 5 URLs only shown in dev mode
	}
} 