package analyzer

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

type MemStats struct {
	HeapAlloc    uint64
	TotalAlloc   uint64
	Sys          uint64
	NumGC        uint32
	PauseTotalNs uint64
}

func getMemStats() MemStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return MemStats{
		HeapAlloc:    stats.HeapAlloc,
		TotalAlloc:   stats.TotalAlloc,
		Sys:          stats.Sys,
		NumGC:        stats.NumGC,
		PauseTotalNs: stats.PauseTotalNs,
	}
}

func printMemStats(t *testing.T, before, after MemStats) {
	t.Logf("Memory Statistics:")
	t.Logf("Heap Allocation: %d bytes -> %d bytes (Delta: %d bytes)", 
		before.HeapAlloc, after.HeapAlloc, after.HeapAlloc-before.HeapAlloc)
	t.Logf("Total Allocation: %d bytes -> %d bytes (Delta: %d bytes)", 
		before.TotalAlloc, after.TotalAlloc, after.TotalAlloc-before.TotalAlloc)
	t.Logf("System Memory: %d bytes -> %d bytes (Delta: %d bytes)", 
		before.Sys, after.Sys, after.Sys-before.Sys)
	t.Logf("Number of GC runs: %d -> %d (Delta: %d)", 
		before.NumGC, after.NumGC, after.NumGC-before.NumGC)
	t.Logf("Total GC Pause: %d ns -> %d ns (Delta: %d ns)", 
		before.PauseTotalNs, after.PauseTotalNs, after.PauseTotalNs-before.PauseTotalNs)
}

func TestMemoryEfficiency(t *testing.T) {
	// Test URLs with different characteristics
	urls := []string{
		"https://www.example.com",
		"https://www.google.com",
		"https://www.github.com",
		"https://www.wikipedia.org",
		"https://www.reddit.com",
	}

	// Create analyzer instance
	analyzer := New()

	// Force garbage collection before starting
	runtime.GC()
	time.Sleep(time.Second) // Let GC complete

	// Get initial memory stats
	before := getMemStats()

	// Number of iterations for each URL
	iterations := 10
	// Number of concurrent requests
	concurrency := 5

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Track cache statistics
	var totalRequests, cacheHits int
	var statsMutex sync.Mutex

	// Run load test
	for i := 0; i < iterations; i++ {
		for _, url := range urls {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Check if context is cancelled
				select {
				case <-ctx.Done():
					t.Logf("Context cancelled, stopping test")
					return
				default:
				}

				// Check cache first
				isCached := analyzer.IsCached(url)
				
				// Analyze URL
				_, err := analyzer.Analyze(url)
				if err != nil {
					t.Logf("Error analyzing URL %s: %v", url, err)
					return
				}

				// Update statistics
				statsMutex.Lock()
				totalRequests++
				if isCached {
					cacheHits++
				}
				statsMutex.Unlock()
			}(url)
		}
	}

	// Wait for all requests to complete
	wg.Wait()

	// Force garbage collection after test
	runtime.GC()
	time.Sleep(time.Second) // Let GC complete

	// Get final memory stats
	after := getMemStats()

	// Print memory statistics
	printMemStats(t, before, after)

	// Print cache statistics
	stats := analyzer.GetCacheStats()
	t.Logf("\nCache Statistics:")
	t.Logf("Analysis Cache Entries: %d", stats.AnalysisEntries)
	t.Logf("Link Cache Entries: %d", stats.LinkEntries)
	t.Logf("Analysis Cache Hits: %d", stats.AnalysisCacheHits)
	t.Logf("Link Cache Hits: %d", stats.LinkCacheHits)
	t.Logf("Analysis Cache Misses: %d", stats.AnalysisCacheMisses)
	t.Logf("Link Cache Misses: %d", stats.LinkCacheMisses)
	t.Logf("Cache Hit Rate: %.2f%%", float64(cacheHits)/float64(totalRequests)*100)

	// Check for memory leaks
	if after.HeapAlloc > before.HeapAlloc*2 {
		t.Errorf("Possible memory leak: Heap allocation more than doubled")
	}

	// Check if cache size is within expected bounds
	expectedCacheSize := len(urls)
	if stats.AnalysisEntries > expectedCacheSize*2 {
		t.Errorf("Cache size larger than expected: got %d entries, expected maximum %d", 
			stats.AnalysisEntries, expectedCacheSize*2)
	}
}

func TestCachePurging(t *testing.T) {
	analyzer := New()
	
	// Set a very short TTL for testing
	analyzer.SetCacheTTL(1 * time.Second)
	
	// Analyze a URL
	url := "https://www.example.com"
	_, err := analyzer.Analyze(url)
	if err != nil {
		t.Fatalf("Failed to analyze URL: %v", err)
	}
	
	// Verify it's cached
	if !analyzer.IsCached(url) {
		t.Error("URL should be cached immediately after analysis")
	}
	
	// Wait for TTL to expire
	time.Sleep(2 * time.Second)
	
	// Verify it's no longer cached
	if analyzer.IsCached(url) {
		t.Error("URL should not be cached after TTL expiration")
	}
	
	// Get cache stats
	stats := analyzer.GetCacheStats()
	t.Logf("Cache Statistics after TTL expiration:")
	t.Logf("Analysis Cache Entries: %d", stats.AnalysisEntries)
	t.Logf("Analysis Cache Hits: %d", stats.AnalysisCacheHits)
	t.Logf("Analysis Cache Misses: %d", stats.AnalysisCacheMisses)
}

func TestConcurrentCacheAccess(t *testing.T) {
	analyzer := New()
	url := "https://www.example.com"
	
	// Number of concurrent goroutines
	concurrency := 100
	
	var wg sync.WaitGroup
	errChan := make(chan error, concurrency)
	
	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Randomly either read from or write to cache
			if i%2 == 0 {
				_, err := analyzer.Analyze(url)
				if err != nil {
					errChan <- fmt.Errorf("analyze error: %v", err)
				}
			} else {
				analyzer.IsCached(url)
			}
		}()
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)
	
	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent access error: %v", err)
	}
	
	// Get final cache stats
	stats := analyzer.GetCacheStats()
	t.Logf("Cache Statistics after concurrent access:")
	t.Logf("Analysis Cache Entries: %d", stats.AnalysisEntries)
	t.Logf("Analysis Cache Hits: %d", stats.AnalysisCacheHits)
	t.Logf("Analysis Cache Misses: %d", stats.AnalysisCacheMisses)
} 