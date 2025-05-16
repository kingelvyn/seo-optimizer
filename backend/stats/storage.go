package stats

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// MonthlyStats represents statistics for a specific month
type MonthlyStats struct {
	// Cache statistics
	AnalysisCacheHits   int            `json:"analysis_hits"`
	AnalysisCacheMisses int            `json:"analysis_misses"`
	LinkCacheHits       int            `json:"link_hits"`
	LinkCacheMisses     int            `json:"link_misses"`
	
	// General statistics
	UniqueVisitors      map[string]time.Time `json:"unique_visitors"`
	AnalysisRequests    int                  `json:"analysis_requests"`
	ErrorCount          int                  `json:"error_count"`
	PopularUrls         map[string]int       `json:"popular_urls"`
	TotalLoadTime       float64              `json:"total_load_time"`
	TotalRequests       int                  `json:"total_requests"`
	
	// Metadata
	LastUpdated         time.Time            `json:"last_updated"`
}

// Storage handles persistent storage of statistics
type Storage struct {
	mutex       sync.RWMutex
	stats       map[string]*MonthlyStats // key: "YYYY-MM"
	filePath    string
	lastWrite   time.Time
	writeBuffer chan struct{}
}

// NewStorage creates a new statistics storage instance
func NewStorage(dataDir string) (*Storage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "stats.json")
	s := &Storage{
		stats:       make(map[string]*MonthlyStats),
		filePath:    filePath,
		writeBuffer: make(chan struct{}, 1), // Buffer for write requests
	}

	// Load existing stats if file exists
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load stats: %w", err)
	}

	// Try to migrate old statistics
	if err := s.migrateOldStats(dataDir); err != nil {
		log.Printf("Warning: Failed to migrate old statistics: %v", err)
	}

	// Start background writer
	go s.backgroundWriter()

	return s, nil
}

// migrateOldStats attempts to migrate statistics from the old format
func (s *Storage) migrateOldStats(dataDir string) error {
	oldStatsPath := filepath.Join(dataDir, "statistics.json")
	data, err := os.ReadFile(oldStatsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No old stats to migrate
		}
		return err
	}

	// Parse old statistics
	var oldStats struct {
		UniqueVisitors   map[string]time.Time `json:"uniqueVisitors"`
		AnalysisRequests int                  `json:"analysisRequests"`
		ErrorCount       int                  `json:"errorCount"`
		PopularUrls      map[string]int       `json:"popularUrls"`
		AverageLoadTime  float64              `json:"averageLoadTime"`
		TotalRequests    int                  `json:"totalRequests"`
		LastPersisted    time.Time            `json:"lastPersisted"`
		// Add cache-related fields
		AnalysisCacheHits   int `json:"analysisCacheHits"`
		AnalysisCacheMisses int `json:"analysisCacheMisses"`
		LinkCacheHits       int `json:"linkCacheHits"`
		LinkCacheMisses     int `json:"linkCacheMisses"`
	}

	if err := json.Unmarshal(data, &oldStats); err != nil {
		return err
	}

	// Initialize maps if they're nil
	if oldStats.UniqueVisitors == nil {
		oldStats.UniqueVisitors = make(map[string]time.Time)
	}
	if oldStats.PopularUrls == nil {
		oldStats.PopularUrls = make(map[string]int)
	}

	// Get current month's stats
	month := getCurrentMonth()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	stats, exists := s.stats[month]
	if !exists {
		stats = &MonthlyStats{
			UniqueVisitors: make(map[string]time.Time),
			PopularUrls:    make(map[string]int),
		}
		s.stats[month] = stats
	} else {
		// Initialize maps if they're nil
		if stats.UniqueVisitors == nil {
			stats.UniqueVisitors = make(map[string]time.Time)
		}
		if stats.PopularUrls == nil {
			stats.PopularUrls = make(map[string]int)
		}
	}

	// Migrate data - preserve existing values if they exist
	for ip, timestamp := range oldStats.UniqueVisitors {
		if _, exists := stats.UniqueVisitors[ip]; !exists {
			stats.UniqueVisitors[ip] = timestamp
		}
	}
	for url, count := range oldStats.PopularUrls {
		stats.PopularUrls[url] += count // Add to existing count if any
	}
	
	// Preserve existing counters by adding old values
	stats.AnalysisRequests += oldStats.AnalysisRequests
	stats.ErrorCount += oldStats.ErrorCount
	stats.TotalLoadTime += oldStats.AverageLoadTime * float64(oldStats.TotalRequests)
	stats.TotalRequests += oldStats.TotalRequests
	
	// Preserve cache statistics
	stats.AnalysisCacheHits += oldStats.AnalysisCacheHits
	stats.AnalysisCacheMisses += oldStats.AnalysisCacheMisses
	stats.LinkCacheHits += oldStats.LinkCacheHits
	stats.LinkCacheMisses += oldStats.LinkCacheMisses

	// Update last updated timestamp if needed
	if oldStats.LastPersisted.After(stats.LastUpdated) {
		stats.LastUpdated = oldStats.LastPersisted
	}

	// Request immediate write of migrated data
	s.requestWrite()

	// Rename old statistics file to prevent re-migration
	backupPath := oldStatsPath + ".bak"
	return os.Rename(oldStatsPath, backupPath)
}

// TrackVisitor records a unique visitor
func (s *Storage) TrackVisitor(ip string) {
	month := getCurrentMonth()
	
	s.mutex.Lock()
	defer s.mutex.Unlock()

	stats, exists := s.stats[month]
	if !exists {
		stats = &MonthlyStats{
			UniqueVisitors: make(map[string]time.Time),
			PopularUrls:    make(map[string]int),
		}
		s.stats[month] = stats
	}

	stats.UniqueVisitors[ip] = time.Now()
	stats.LastUpdated = time.Now()

	if time.Since(s.lastWrite) > time.Minute {
		s.requestWrite()
		s.lastWrite = time.Now()
	}
}

// TrackAnalysis records an analysis request
func (s *Storage) TrackAnalysis(url string, loadTime float64, isError bool) {
	month := getCurrentMonth()
	
	s.mutex.Lock()
	defer s.mutex.Unlock()

	stats, exists := s.stats[month]
	if !exists {
		stats = &MonthlyStats{
			UniqueVisitors: make(map[string]time.Time),
			PopularUrls:    make(map[string]int),
		}
		s.stats[month] = stats
	}

	stats.AnalysisRequests++
	stats.TotalRequests++
	stats.TotalLoadTime += loadTime
	if isError {
		stats.ErrorCount++
	}
	stats.PopularUrls[url]++
	stats.LastUpdated = time.Now()

	if time.Since(s.lastWrite) > time.Minute {
		s.requestWrite()
		s.lastWrite = time.Now()
	}
}

// load reads statistics from file
func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create temporary map for loading
	tempStats := make(map[string]*MonthlyStats)
	if err := json.Unmarshal(data, &tempStats); err != nil {
		return err
	}

	// Ensure all maps are properly initialized
	for month, stats := range tempStats {
		if stats.UniqueVisitors == nil {
			stats.UniqueVisitors = make(map[string]time.Time)
		}
		if stats.PopularUrls == nil {
			stats.PopularUrls = make(map[string]int)
		}

		// Preserve any existing data by merging
		if existingStats, exists := s.stats[month]; exists {
			// Merge unique visitors
			for ip, timestamp := range existingStats.UniqueVisitors {
				if _, ok := stats.UniqueVisitors[ip]; !ok {
					stats.UniqueVisitors[ip] = timestamp
				}
			}
			// Merge popular URLs
			for url, count := range existingStats.PopularUrls {
				stats.PopularUrls[url] += count
			}
			// Add counters
			stats.AnalysisRequests += existingStats.AnalysisRequests
			stats.ErrorCount += existingStats.ErrorCount
			stats.TotalLoadTime += existingStats.TotalLoadTime
			stats.TotalRequests += existingStats.TotalRequests
			stats.AnalysisCacheHits += existingStats.AnalysisCacheHits
			stats.AnalysisCacheMisses += existingStats.AnalysisCacheMisses
			stats.LinkCacheHits += existingStats.LinkCacheHits
			stats.LinkCacheMisses += existingStats.LinkCacheMisses

			// Keep the most recent last updated time
			if existingStats.LastUpdated.After(stats.LastUpdated) {
				stats.LastUpdated = existingStats.LastUpdated
			}
		}
	}

	// Replace the storage's stats with the merged data
	s.stats = tempStats
	return nil
}

// save writes statistics to file
func (s *Storage) save() error {
	s.mutex.RLock()
	data, err := json.Marshal(s.stats)
	s.mutex.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	// Write to temporary file first
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tempFile, s.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file if rename fails
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// backgroundWriter handles periodic writes to disk
func (s *Storage) backgroundWriter() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.writeBuffer:
			// Immediate write requested
			s.save()
		case <-ticker.C:
			// Periodic write
			s.save()
		}
	}
}

// getCurrentMonth returns the current month key in YYYY-MM format
func getCurrentMonth() string {
	return time.Now().Format("2006-01")
}

// requestWrite signals that a write to disk is needed
func (s *Storage) requestWrite() {
	select {
	case s.writeBuffer <- struct{}{}:
		// Write requested
	default:
		// Buffer full, write already pending
	}
}

// IncrementStats increments the specified statistics
func (s *Storage) IncrementStats(analysisHits, analysisMisses, linkHits, linkMisses int) {
	month := getCurrentMonth()
	
	s.mutex.Lock()
	defer s.mutex.Unlock()

	stats, exists := s.stats[month]
	if !exists {
		stats = &MonthlyStats{
			UniqueVisitors: make(map[string]time.Time),
			PopularUrls:    make(map[string]int),
		}
		s.stats[month] = stats
	} else {
		// Ensure maps are initialized
		if stats.UniqueVisitors == nil {
			stats.UniqueVisitors = make(map[string]time.Time)
		}
		if stats.PopularUrls == nil {
			stats.PopularUrls = make(map[string]int)
		}
	}

	stats.AnalysisCacheHits += analysisHits
	stats.AnalysisCacheMisses += analysisMisses
	stats.LinkCacheHits += linkHits
	stats.LinkCacheMisses += linkMisses
	stats.LastUpdated = time.Now()

	// Request a write if enough time has passed
	if time.Since(s.lastWrite) > time.Minute {
		s.requestWrite()
		s.lastWrite = time.Now()
	}
}

// GetCurrentStats returns statistics for the current month
func (s *Storage) GetCurrentStats() MonthlyStats {
	month := getCurrentMonth()
	
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if stats, exists := s.stats[month]; exists {
		// Ensure maps are initialized
		if stats.UniqueVisitors == nil {
			stats.UniqueVisitors = make(map[string]time.Time)
		}
		if stats.PopularUrls == nil {
			stats.PopularUrls = make(map[string]int)
		}
		return *stats
	}
	return MonthlyStats{
		UniqueVisitors: make(map[string]time.Time),
		PopularUrls:    make(map[string]int),
	}
}

// Cleanup removes statistics older than the specified number of months
func (s *Storage) Cleanup(retainMonths int) {
	currentTime := time.Now()
	currentMonth := currentTime.Format("2006-01")
	
	// Calculate previous month
	previousMonth := currentTime.AddDate(0, -1, 0).Format("2006-01")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Only keep current and previous month
	for key := range s.stats {
		if key != currentMonth && key != previousMonth {
			delete(s.stats, key)
		}
	}

	// Request a write to persist changes
	s.requestWrite()
	
	// Log retained months for debugging
	log.Printf("Retained statistics for months: %s, %s", currentMonth, previousMonth)
}

// GetMonthlyStats returns statistics for a specific month
func (s *Storage) GetMonthlyStats(yearMonth string) (MonthlyStats, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if stats, exists := s.stats[yearMonth]; exists {
		return *stats, true
	}
	return MonthlyStats{}, false
}

// GetAllMonths returns a sorted list of all months that have statistics
func (s *Storage) GetAllMonths() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	months := make([]string, 0, len(s.stats))
	for month := range s.stats {
		months = append(months, month)
	}
	
	// Sort months in descending order (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(months)))
	
	return months
} 