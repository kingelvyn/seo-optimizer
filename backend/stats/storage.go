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
	AnalysisCacheHits   int `json:"analysis_hits"`
	AnalysisCacheMisses int `json:"analysis_misses"`
	LinkCacheHits       int `json:"link_hits"`
	LinkCacheMisses     int `json:"link_misses"`
	LastUpdated         time.Time `json:"last_updated"`
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

	// Start background writer
	go s.backgroundWriter()

	return s, nil
}

// load reads statistics from file
func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return json.Unmarshal(data, &s.stats)
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
		stats = &MonthlyStats{}
		s.stats[month] = stats
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
		return *stats
	}
	return MonthlyStats{}
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