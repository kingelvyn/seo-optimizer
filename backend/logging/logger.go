package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Environment variable name for controlling statistics visibility
const (
	ENV_DEV_MODE = "DEV_MODE"
	STATS_FILE   = "data/statistics.json"
)

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
		log.Println("Initializing statistics...")
		stats = &Statistics{
			UniqueVisitors:   make(map[string]time.Time),
			PopularURLs:      make(map[string]int),
			LastPersisted:    time.Now(),
		}
		
		// Try to load existing statistics
		if err := stats.Load(); err != nil {
			log.Printf("Could not load existing statistics: %v\n", err)
		}

		// Log current mode and statistics state
		log.Printf("DEV_MODE: %s\n", os.Getenv(ENV_DEV_MODE))
		log.Printf("Initial stats state: %+v\n", stats.GetStatistics())
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
		log.Printf("Tracked URL: %s (Count: %d)\n", cleanedURL, s.PopularURLs[cleanedURL])
	}
	
	if hasError {
		s.ErrorCount++
	}
	
	// Update average load time
	s.TotalLoadTime += loadTime
	s.RequestCount++
	s.AverageLoadTime = s.TotalLoadTime / float64(s.RequestCount)

	// Save statistics periodically
	if s.AnalysisRequests%10 == 0 { // Save every 10 requests
		if err := s.Save(); err != nil {
			log.Printf("Error saving statistics: %v\n", err)
		}
	}
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
	
	// Ensure the data directory exists
	if err := os.MkdirAll(filepath.Dir(STATS_FILE), 0755); err != nil {
		return fmt.Errorf("could not create data directory: %v", err)
	}
	
	file, err := os.Create(STATS_FILE)
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
	file, err := os.Open(STATS_FILE)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with empty statistics if file doesn't exist
			s.UniqueVisitors = make(map[string]time.Time)
			s.PopularURLs = make(map[string]int)
			s.LastPersisted = time.Now()
			return nil
		}
		return fmt.Errorf("could not open statistics file: %v", err)
	}
	defer file.Close()
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(s); err != nil {
		// If there's an error decoding, initialize with empty statistics
		s.UniqueVisitors = make(map[string]time.Time)
		s.PopularURLs = make(map[string]int)
		s.LastPersisted = time.Now()
		return fmt.Errorf("could not decode statistics (initializing empty): %v", err)
	}
	
	return nil
}

// GetStatistics returns a copy of the current statistics
func (s *Statistics) GetStatistics() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Basic statistics always included
	stats := map[string]interface{}{
		"uniqueVisitors24h": s.GetUniqueVisitorsCount(),
		"totalRequests":     s.AnalysisRequests,
		"errorRate":         s.GetErrorRate(),
		"averageLoadTime":   s.AverageLoadTime,
	}

	// Add popular URLs in development mode
	if os.Getenv(ENV_DEV_MODE) == "true" {
		log.Println("Development mode: Including popular URLs in statistics")
		popularURLs := s.GetPopularURLs(5)
		if len(popularURLs) > 0 {
			stats["popularUrls"] = popularURLs
		} else {
			log.Println("No popular URLs available")
		}
	}

	return stats
} 