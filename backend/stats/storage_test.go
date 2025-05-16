package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "stats-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create new storage
	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test incrementing stats
	t.Run("IncrementStats", func(t *testing.T) {
		storage.IncrementStats(1, 2, 3, 4)
		stats := storage.GetCurrentStats()

		if stats.AnalysisCacheHits != 1 {
			t.Errorf("Expected 1 analysis hit, got %d", stats.AnalysisCacheHits)
		}
		if stats.AnalysisCacheMisses != 2 {
			t.Errorf("Expected 2 analysis misses, got %d", stats.AnalysisCacheMisses)
		}
		if stats.LinkCacheHits != 3 {
			t.Errorf("Expected 3 link hits, got %d", stats.LinkCacheHits)
		}
		if stats.LinkCacheMisses != 4 {
			t.Errorf("Expected 4 link misses, got %d", stats.LinkCacheMisses)
		}
	})

	// Test persistence
	t.Run("Persistence", func(t *testing.T) {
		// Force a save
		storage.requestWrite()
		time.Sleep(100 * time.Millisecond) // Give time for the write to complete

		// Create new storage instance pointing to same directory
		storage2, err := NewStorage(tempDir)
		if err != nil {
			t.Fatalf("Failed to create second storage: %v", err)
		}

		stats := storage2.GetCurrentStats()
		if stats.AnalysisCacheHits != 1 {
			t.Errorf("Expected 1 analysis hit after reload, got %d", stats.AnalysisCacheHits)
		}
	})

	// Test cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Add some old stats
		oldMonth := time.Now().AddDate(0, -2, 0).Format("2006-01")
		storage.stats[oldMonth] = &MonthlyStats{
			AnalysisCacheHits: 100,
			LastUpdated:       time.Now().AddDate(0, -2, 0),
		}

		// Run cleanup keeping only 1 month of data
		storage.Cleanup(1)

		// Verify old stats are gone
		if _, exists := storage.stats[oldMonth]; exists {
			t.Error("Old stats should have been cleaned up")
		}
	})

	// Test file size
	t.Run("FileSize", func(t *testing.T) {
		// Force a save
		storage.requestWrite()
		time.Sleep(100 * time.Millisecond) // Give time for the write to complete

		// Check file size
		info, err := os.Stat(filepath.Join(tempDir, "stats.json"))
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		// File should be relatively small (< 1KB for this test data)
		if info.Size() > 1024 {
			t.Errorf("File size too large: %d bytes", info.Size())
		}
	})

	// Test concurrent access
	t.Run("ConcurrentAccess", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					storage.IncrementStats(1, 1, 1, 1)
					storage.GetCurrentStats()
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify final counts
		stats := storage.GetCurrentStats()
		expectedCount := 1000 // 10 goroutines * 100 iterations
		totalHits := stats.AnalysisCacheHits + stats.LinkCacheHits
		if totalHits != expectedCount*2 {
			t.Errorf("Expected %d total hits, got %d", expectedCount*2, totalHits)
		}
	})
} 