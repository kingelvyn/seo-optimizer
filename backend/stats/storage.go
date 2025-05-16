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

// NewMonthlyStats creates a new MonthlyStats instance with initialized maps
func NewMonthlyStats() *MonthlyStats {
	return &MonthlyStats{
		UniqueVisitors: make(map[string]time.Time),
		PopularUrls:    make(map[string]int),
		LastUpdated:    time.Now(),
	}
}

// Storage handles persistent storage of statistics
type Storage struct {
	mutex       sync.RWMutex
	stats       map[string]*MonthlyStats // key: "YYYY-MM"
	filePath    string
	lastWrite   time.Time
	writeBuffer chan struct{}
	done        chan struct{} // Channel to signal shutdown
}

// NewStorage creates a new statistics storage instance
func NewStorage(dataDir string) (*Storage, error) {
	log.Printf("Initializing storage with data directory: %s", dataDir)
	
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "stats.json")
	s := &Storage{
		stats:       make(map[string]*MonthlyStats),
		filePath:    filePath,
		writeBuffer: make(chan struct{}, 1),
		done:        make(chan struct{}),
	}

	// Initialize current month's stats
	currentMonth := getCurrentMonth()
	s.stats[currentMonth] = NewMonthlyStats()
	log.Printf("Initialized current month stats: %s", currentMonth)

	// Load existing stats if file exists
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading existing stats: %v", err)
			return nil, fmt.Errorf("failed to load stats: %w", err)
		}
		log.Printf("No existing stats file found, starting fresh")
	} else {
		log.Printf("Successfully loaded existing stats")
	}

	// Try to migrate old statistics
	if err := s.migrateOldStats(dataDir); err != nil {
		log.Printf("Warning: Failed to migrate old statistics: %v", err)
	}

	// Force an immediate save to ensure everything is written
	if err := s.save(); err != nil {
		log.Printf("Warning: Failed to perform initial stats save: %v", err)
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
	if s == nil {
		log.Printf("ERROR: Storage is nil in TrackVisitor")
		return
	}
	if ip == "" {
		log.Printf("WARNING: Empty IP address in TrackVisitor")
		return
	}

	month := getCurrentMonth()
	
	// Check existence under read lock
	s.mutex.RLock()
	stats, exists := s.stats[month]
	s.mutex.RUnlock()

	if !exists {
		s.mutex.Lock()
		stats = NewMonthlyStats()
		s.stats[month] = stats
		s.mutex.Unlock()
	}

	// Update visitor under write lock
	s.mutex.Lock()
	stats.UniqueVisitors[ip] = time.Now()
	stats.LastUpdated = time.Now()
	s.mutex.Unlock()

	// Get count under read lock
	s.mutex.RLock()
	visitorCount := len(stats.UniqueVisitors)
	s.mutex.RUnlock()

	log.Printf("Tracked visitor IP: %s, total unique visitors: %d", ip, visitorCount)

	// Check write timing under read lock
	s.mutex.RLock()
	shouldWrite := time.Since(s.lastWrite) > time.Minute
	s.mutex.RUnlock()

	if shouldWrite {
		s.mutex.Lock()
		s.lastWrite = time.Now()
		s.mutex.Unlock()
		s.requestWrite()
	}
}

// TrackAnalysis records an analysis request
func (s *Storage) TrackAnalysis(url string, loadTime float64, isError bool) {
	if s == nil {
		log.Printf("ERROR: Storage is nil in TrackAnalysis")
		return
	}

	month := getCurrentMonth()
	
	// Use shorter lock duration for checking existence
	s.mutex.RLock()
	stats, exists := s.stats[month]
	s.mutex.RUnlock()

	if !exists {
		s.mutex.Lock()
		stats = NewMonthlyStats()
		s.stats[month] = stats
		s.mutex.Unlock()
	}

	// Update stats under a short-lived lock
	s.mutex.Lock()
	stats.AnalysisRequests++
	stats.TotalRequests++
	stats.TotalLoadTime += loadTime
	if isError {
		stats.ErrorCount++
	}
	if url != "" {
		stats.PopularUrls[url]++
	}
	stats.LastUpdated = time.Now()
	s.mutex.Unlock()

	log.Printf("Updated stats after analysis for %s: requests=%d, total=%d, errors=%d", 
		url, stats.AnalysisRequests, stats.TotalRequests, stats.ErrorCount)

	// Check write timing under a short lock
	s.mutex.RLock()
	shouldWrite := time.Since(s.lastWrite) > time.Minute
	s.mutex.RUnlock()

	if shouldWrite {
		s.mutex.Lock()
		s.lastWrite = time.Now()
		s.mutex.Unlock()
		s.requestWrite()
	}
}

// load reads statistics from file
func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		log.Printf("Error reading stats file: %v", err)
		return err
	}

	log.Printf("Loading stats from file: %s", string(data))

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create temporary map for loading
	tempStats := make(map[string]*MonthlyStats)
	if err := json.Unmarshal(data, &tempStats); err != nil {
		log.Printf("Error unmarshaling stats: %v", err)
		return err
	}

	log.Printf("Loaded stats before initialization: %+v", tempStats)

	// Ensure all maps are properly initialized
	for month, stats := range tempStats {
		if stats.UniqueVisitors == nil {
			stats.UniqueVisitors = make(map[string]time.Time)
		}
		if stats.PopularUrls == nil {
			stats.PopularUrls = make(map[string]int)
		}

		log.Printf("Processing month %s, stats before merge: %+v", month, stats)

		// Preserve any existing data by merging
		if existingStats, exists := s.stats[month]; exists {
			log.Printf("Found existing stats for month %s: %+v", month, existingStats)
			
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

			log.Printf("Stats after merge for month %s: %+v", month, stats)
		}
	}

	// Replace the storage's stats with the merged data
	s.stats = tempStats
	log.Printf("Final loaded stats: %+v", s.stats)
	return nil
}

// save writes statistics to file
func (s *Storage) save() error {
	// Create a copy of stats under read lock
	s.mutex.RLock()
	statsCopy := make(map[string]*MonthlyStats)
	for month, stats := range s.stats {
		statsCopy[month] = &MonthlyStats{
			AnalysisCacheHits:   stats.AnalysisCacheHits,
			AnalysisCacheMisses: stats.AnalysisCacheMisses,
			LinkCacheHits:       stats.LinkCacheHits,
			LinkCacheMisses:     stats.LinkCacheMisses,
			AnalysisRequests:    stats.AnalysisRequests,
			ErrorCount:          stats.ErrorCount,
			TotalLoadTime:       stats.TotalLoadTime,
			TotalRequests:       stats.TotalRequests,
			LastUpdated:         stats.LastUpdated,
			UniqueVisitors:      make(map[string]time.Time),
			PopularUrls:         make(map[string]int),
		}
		for k, v := range stats.UniqueVisitors {
			statsCopy[month].UniqueVisitors[k] = v
		}
		for k, v := range stats.PopularUrls {
			statsCopy[month].PopularUrls[k] = v
		}
	}
	s.mutex.RUnlock()

	// Marshal the copy
	data, err := json.Marshal(statsCopy)
	if err != nil {
		log.Printf("Error marshaling stats: %v", err)
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	// Write to temporary file first
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		log.Printf("Error writing temp file: %v", err)
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tempFile, s.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file if rename fails
		log.Printf("Error renaming temp file: %v", err)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	log.Printf("Successfully saved stats to %s", s.filePath)
	return nil
}

// backgroundWriter handles periodic writes to disk
func (s *Storage) backgroundWriter() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.writeBuffer:
			// Immediate write requested
			if err := s.save(); err != nil {
				log.Printf("Error during immediate stats write: %v", err)
			}
		case <-ticker.C:
			// Periodic write
			if err := s.save(); err != nil {
				log.Printf("Error during periodic stats write: %v", err)
			}
		case <-s.done:
			// Final write before shutdown
			log.Printf("Performing final stats write before shutdown")
			if err := s.save(); err != nil {
				log.Printf("Error during final stats write: %v", err)
			}
			return
		}
	}
}

// getCurrentMonth returns the current month key in YYYY-MM format
func getCurrentMonth() string {
	return time.Now().Format("2006-01")
}

// requestWrite signals that a write to disk is needed
func (s *Storage) requestWrite() {
	// Try to write immediately first
	if err := s.save(); err != nil {
		log.Printf("Error during direct stats write: %v", err)
		// Fall back to buffered write if immediate write fails
		select {
		case s.writeBuffer <- struct{}{}:
			log.Printf("Queued stats write after failed direct write")
		default:
			// Try an immediate write again if buffer is full
			if err := s.save(); err != nil {
				log.Printf("Error during retry stats write: %v", err)
			}
		}
	}
}

// IncrementStats increments the specified statistics
func (s *Storage) IncrementStats(analysisHits, analysisMisses, linkHits, linkMisses int) {
	if s == nil {
		log.Printf("ERROR: Storage is nil in IncrementStats")
		return
	}

	month := getCurrentMonth()
	
	// Check existence under read lock
	s.mutex.RLock()
	stats, exists := s.stats[month]
	s.mutex.RUnlock()

	if !exists {
		s.mutex.Lock()
		stats = NewMonthlyStats()
		s.stats[month] = stats
		s.mutex.Unlock()
	}

	// Update stats under write lock
	s.mutex.Lock()
	stats.AnalysisCacheHits += analysisHits
	stats.AnalysisCacheMisses += analysisMisses
	stats.LinkCacheHits += linkHits
	stats.LinkCacheMisses += linkMisses
	stats.LastUpdated = time.Now()
	s.mutex.Unlock()

	log.Printf("Updated cache stats: hits=%d/%d, misses=%d/%d", 
		stats.AnalysisCacheHits, stats.LinkCacheHits,
		stats.AnalysisCacheMisses, stats.LinkCacheMisses)

	// Check write timing under read lock
	s.mutex.RLock()
	shouldWrite := time.Since(s.lastWrite) > time.Minute
	s.mutex.RUnlock()

	if shouldWrite {
		s.mutex.Lock()
		s.lastWrite = time.Now()
		s.mutex.Unlock()
		s.requestWrite()
	}
}

// GetCurrentStats returns statistics for the current month
func (s *Storage) GetCurrentStats() MonthlyStats {
	if s == nil {
		log.Printf("ERROR: Storage is nil in GetCurrentStats")
		return *NewMonthlyStats()
	}

	month := getCurrentMonth()
	
	s.mutex.RLock()
	stats, exists := s.stats[month]
	s.mutex.RUnlock()

	if !exists {
		return *NewMonthlyStats()
	}

	// Make a copy under a short-lived read lock
	s.mutex.RLock()
	statsCopy := MonthlyStats{
		AnalysisCacheHits:   stats.AnalysisCacheHits,
		AnalysisCacheMisses: stats.AnalysisCacheMisses,
		LinkCacheHits:       stats.LinkCacheHits,
		LinkCacheMisses:     stats.LinkCacheMisses,
		AnalysisRequests:    stats.AnalysisRequests,
		ErrorCount:          stats.ErrorCount,
		TotalLoadTime:       stats.TotalLoadTime,
		TotalRequests:       stats.TotalRequests,
		LastUpdated:         stats.LastUpdated,
		UniqueVisitors:      make(map[string]time.Time, len(stats.UniqueVisitors)),
		PopularUrls:         make(map[string]int, len(stats.PopularUrls)),
	}
	s.mutex.RUnlock()

	// Copy maps under separate short-lived locks to minimize contention
	s.mutex.RLock()
	for k, v := range stats.UniqueVisitors {
		statsCopy.UniqueVisitors[k] = v
	}
	s.mutex.RUnlock()

	s.mutex.RLock()
	for k, v := range stats.PopularUrls {
		statsCopy.PopularUrls[k] = v
	}
	s.mutex.RUnlock()

	return statsCopy
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

// Shutdown ensures all statistics are written before the application exits
func (s *Storage) Shutdown() error {
	if s == nil {
		return nil
	}

	log.Printf("Shutting down statistics storage")
	
	// Signal the background writer to stop and perform final write
	close(s.done)

	// Perform one final write directly
	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save stats during shutdown: %w", err)
	}

	log.Printf("Statistics storage shutdown complete")
	return nil
} 