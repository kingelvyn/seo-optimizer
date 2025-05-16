package analyzer

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/seo-optimizer/backend/stats"
)

// Object pools for frequently allocated objects
var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	
	urlSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 100)
		},
	}
	
	mapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]bool, 100)
		},
	}
	
	analysisPool = sync.Pool{
		New: func() interface{} {
			return &SEOAnalysis{
				Content: ContentAnalysis{
					KeywordDensity: make(map[string]float64),
				},
				Headers: HeaderAnalysis{
					H1Text: make([]string, 0, 5),
				},
			}
		},
	}
)

// Cache entry with expiration
type cacheEntry struct {
	analysis  *SEOAnalysis
	timestamp time.Time
}

// CacheStats provides statistics about the analyzer's cache
type CacheStats struct {
	AnalysisEntries     int           `json:"analysisEntries"`
	LinkEntries         int           `json:"linkEntries"`
	AnalysisCacheHits   int           `json:"analysisCacheHits"`
	LinkCacheHits       int           `json:"linkCacheHits"`
	AnalysisCacheMisses int           `json:"analysisCacheMisses"`
	LinkCacheMisses     int           `json:"linkCacheMisses"`
	AnalysisCacheTTL    time.Duration `json:"analysisCacheTTL"`
	LinkCacheTTL        time.Duration `json:"linkCacheTTL"`
}

// Analyzer performs SEO analysis on a given URL
type Analyzer struct {
	client            *http.Client
	cache             map[string]cacheEntry
	cacheMutex        sync.RWMutex
	cacheTTL          time.Duration
	linkCache         map[string]linkCacheEntry
	linkCacheMutex    sync.RWMutex
	linkCacheTTL      time.Duration
	maxCacheSize      int
	maxLinkCacheSize  int
	lastCleanup       time.Time
	cleanupInterval   time.Duration
	stats             *stats.Storage
}

// Link cache entry
type linkCacheEntry struct {
	accessible bool
	timestamp  time.Time
}

// New creates a new Analyzer instance
func New(dataDir string) (*Analyzer, error) {
	// Create an optimized HTTP client with:
	// - Reasonable timeout
	// - Connection pooling
	// - Keep-alive connections
	transport := &http.Transport{
		MaxIdleConns:        100,              // Increase from default 2
		MaxIdleConnsPerHost: 10,               // Increase from default 2
		IdleConnTimeout:     90 * time.Second, // Default is 90s
		TLSHandshakeTimeout: 10 * time.Second, // Default is 10s
		DisableCompression:  false,            // Enable compression
	}
	
	// Initialize statistics storage
	statsStorage, err := stats.NewStorage(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stats storage: %w", err)
	}
	
	analyzer := &Analyzer{
		client: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		cache:             make(map[string]cacheEntry),
		cacheTTL:         30 * time.Minute, // Cache results for 30 minutes
		linkCache:        make(map[string]linkCacheEntry),
		linkCacheTTL:     10 * time.Minute, // Cache link status for 10 minutes
		maxCacheSize:     1000,             // Maximum number of cached analyses
		maxLinkCacheSize: 10000,            // Maximum number of cached link statuses
		cleanupInterval:  5 * time.Minute,  // Run cleanup every 5 minutes
		lastCleanup:      time.Now(),
		stats:            statsStorage,
	}
	
	// Start cleanup goroutine
	go analyzer.periodicCleanup()
	
	return analyzer, nil
}

// periodicCleanup removes expired entries from both caches periodically
func (a *Analyzer) periodicCleanup() {
	ticker := time.NewTicker(a.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		a.cleanup()
	}
}

// cleanup removes expired entries and ensures cache size limits
func (a *Analyzer) cleanup() {
	now := time.Now()
	
	// Cleanup analysis cache
	a.cacheMutex.Lock()
	for key, entry := range a.cache {
		if now.Sub(entry.timestamp) > a.cacheTTL {
			delete(a.cache, key)
		}
	}
	
	// If still over size limit, remove oldest entries
	if len(a.cache) > a.maxCacheSize {
		// Convert map to slice for sorting
		entries := make([]struct {
			key       string
			timestamp time.Time
		}, 0, len(a.cache))
		
		for key, entry := range a.cache {
			entries = append(entries, struct {
				key       string
				timestamp time.Time
			}{key, entry.timestamp})
		}
		
		// Sort by timestamp
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].timestamp.Before(entries[j].timestamp)
		})
		
		// Remove oldest entries until under limit
		for i := 0; i < len(entries)-a.maxCacheSize; i++ {
			delete(a.cache, entries[i].key)
		}
	}
	a.cacheMutex.Unlock()
	
	// Cleanup link cache
	a.linkCacheMutex.Lock()
	for key, entry := range a.linkCache {
		if now.Sub(entry.timestamp) > a.linkCacheTTL {
			delete(a.linkCache, key)
		}
	}
	
	// If still over size limit, remove oldest entries
	if len(a.linkCache) > a.maxLinkCacheSize {
		// Convert map to slice for sorting
		entries := make([]struct {
			key       string
			timestamp time.Time
		}, 0, len(a.linkCache))
		
		for key, entry := range a.linkCache {
			entries = append(entries, struct {
				key       string
				timestamp time.Time
			}{key, entry.timestamp})
		}
		
		// Sort by timestamp
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].timestamp.Before(entries[j].timestamp)
		})
		
		// Remove oldest entries until under limit
		for i := 0; i < len(entries)-a.maxLinkCacheSize; i++ {
			delete(a.linkCache, entries[i].key)
		}
	}
	a.linkCacheMutex.Unlock()
	
	a.lastCleanup = now
}

// SetMaxCacheSize sets the maximum number of entries in the analysis cache
func (a *Analyzer) SetMaxCacheSize(size int) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	a.maxCacheSize = size
	a.cleanup() // Run cleanup immediately if new size is smaller
}

// SetMaxLinkCacheSize sets the maximum number of entries in the link cache
func (a *Analyzer) SetMaxLinkCacheSize(size int) {
	a.linkCacheMutex.Lock()
	defer a.linkCacheMutex.Unlock()
	a.maxLinkCacheSize = size
	a.cleanup() // Run cleanup immediately if new size is smaller
}

// SetCacheTTL sets the cache TTL
func (a *Analyzer) SetCacheTTL(ttl time.Duration) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	a.cacheTTL = ttl
}

// ClearCache clears the analysis cache
func (a *Analyzer) ClearCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	a.cache = make(map[string]cacheEntry)
}

// generateCacheKey creates a unique key for the URL
func generateCacheKey(url string) string {
	hash := md5.Sum([]byte(url))
	return hex.EncodeToString(hash[:])
}

// GetCacheStats returns statistics about the cache
func (a *Analyzer) GetCacheStats() CacheStats {
	currentStats := a.stats.GetCurrentStats()
	
	a.cacheMutex.RLock()
	analysisEntries := len(a.cache)
	analysisTTL := a.cacheTTL
	a.cacheMutex.RUnlock()
	
	a.linkCacheMutex.RLock()
	linkEntries := len(a.linkCache)
	linkTTL := a.linkCacheTTL
	a.linkCacheMutex.RUnlock()
	
	return CacheStats{
		AnalysisEntries:     analysisEntries,
		LinkEntries:         linkEntries,
		AnalysisCacheHits:   currentStats.AnalysisCacheHits,
		LinkCacheHits:       currentStats.LinkCacheHits,
		AnalysisCacheMisses: currentStats.AnalysisCacheMisses,
		LinkCacheMisses:     currentStats.LinkCacheMisses,
		AnalysisCacheTTL:    analysisTTL,
		LinkCacheTTL:        linkTTL,
	}
}

// IsCached checks if a URL is in the cache and not expired
func (a *Analyzer) IsCached(url string) bool {
	cacheKey := generateCacheKey(url)
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()
	
	entry, found := a.cache[cacheKey]
	if found && time.Since(entry.timestamp) < a.cacheTTL {
		return true
	}
	return false
}

// Analyze performs a complete SEO analysis of the given URL
func (a *Analyzer) Analyze(url string) (*SEOAnalysis, error) {
	// Check if cleanup is needed
	if time.Since(a.lastCleanup) > a.cleanupInterval {
		go a.cleanup() // Run cleanup in background
	}
	
	// Create a context with timeout for the entire analysis process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Check cache first
	cacheKey := generateCacheKey(url)
	a.cacheMutex.RLock()
	if entry, found := a.cache[cacheKey]; found {
		if time.Since(entry.timestamp) < a.cacheTTL {
			a.stats.IncrementStats(1, 0, 0, 0) // Increment analysis cache hits
			a.cacheMutex.RUnlock()
			return entry.analysis, nil
		}
	}
	a.cacheMutex.RUnlock()
	
	// Not in cache or expired
	a.stats.IncrementStats(0, 1, 0, 0) // Increment analysis cache misses
	
	// Perform analysis
	analysis, err := a.AnalyzeWithContext(ctx, url)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	a.cacheMutex.Lock()
	a.cache[cacheKey] = cacheEntry{
		analysis:  analysis,
		timestamp: time.Now(),
	}
	a.cacheMutex.Unlock()
	
	return analysis, nil
}

// AnalyzeWithContext performs a complete SEO analysis of the given URL with context
func (a *Analyzer) AnalyzeWithContext(ctx context.Context, url string) (*SEOAnalysis, error) {
	startTime := time.Now()

	// Get an analysis object from the pool
	analysis := analysisPool.Get().(*SEOAnalysis)
	analysis.URL = url
	analysis.Content.KeywordDensity = make(map[string]float64)
	analysis.Headers.H1Text = analysis.Headers.H1Text[:0]

	// Create a request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		analysisPool.Put(analysis)
		return nil, err
	}
	
	// Set user agent to avoid being blocked by some websites
	req.Header.Set("User-Agent", "SEOAnalyzer/1.0")

	// Fetch the page
	resp, err := a.client.Do(req)
	if err != nil {
		analysisPool.Put(analysis)
		return nil, err
	}
	defer resp.Body.Close()

	// Get actual page size from response headers if available
	pageSize := 0
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.Atoi(contentLength); err == nil {
			pageSize = size
		}
	}

	// Get a buffer from the pool
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	// Read the response body into the buffer
	if _, err := io.Copy(buf, resp.Body); err != nil {
		analysisPool.Put(analysis)
		return nil, err
	}

	// If we couldn't get the page size from headers, calculate it from the buffer
	if pageSize == 0 {
		pageSize = buf.Len()
	}

	// Parse the HTML from the buffer
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		analysisPool.Put(analysis)
		return nil, err
	}

	// Calculate load time before any processing
	loadTime := time.Since(startTime)

	// Check mobile optimization
	mobileOptimized := false
	doc.Find("meta[name='viewport']").Each(func(_ int, s *goquery.Selection) {
		content, exists := s.Attr("content")
		if exists && strings.Contains(strings.ToLower(content), "width=device-width") {
			mobileOptimized = true
		}
	})

	// Perform analysis with context awareness
	analysis.Title = a.analyzeTitleTag(doc)
	analysis.Meta = a.analyzeMetaTags(doc)
	analysis.Headers = a.analyzeHeaders(doc)
	analysis.Content = a.analyzeContent(doc)
	analysis.Performance = a.analyzePerformance(pageSize, loadTime, mobileOptimized)
	analysis.Links = a.analyzeLinksWithContext(ctx, doc, url)

	// Calculate overall score and recommendations
	analysis.Score = a.calculateOverallScore(analysis)
	analysis.Recommendations = a.generateRecommendations(analysis)

	return analysis, nil
}

func (a *Analyzer) analyzeTitleTag(doc *goquery.Document) TitleAnalysis {
	title := doc.Find("title").First().Text()
	length := len(title)

	score := 0
	if length > 0 {
		if length >= 30 && length <= 60 {
			score = 100
		} else if length < 30 {
			score = 50
		} else {
			score = 70
		}
	}

	return TitleAnalysis{
		Title:    title,
		Length:   length,
		HasTitle: length > 0,
		Score:    score,
	}
}

func (a *Analyzer) analyzeMetaTags(doc *goquery.Document) MetaAnalysis {
	meta := MetaAnalysis{}
	score := 0

	// Description
	meta.Description, _ = doc.Find("meta[name='description']").Attr("content")
	meta.DescriptionLen = len(meta.Description)
	meta.HasDescription = meta.DescriptionLen > 0

	// Keywords
	meta.Keywords, _ = doc.Find("meta[name='keywords']").Attr("content")
	meta.HasKeywords = len(meta.Keywords) > 0

	// Robots
	meta.Robots, _ = doc.Find("meta[name='robots']").Attr("content")

	// Viewport
	meta.Viewport, _ = doc.Find("meta[name='viewport']").Attr("content")

	// Score calculation
	if meta.HasDescription {
		if meta.DescriptionLen >= 120 && meta.DescriptionLen <= 160 {
			score += 40
		} else {
			score += 20
		}
	}
	if meta.HasKeywords {
		score += 20
	}
	if meta.Viewport != "" {
		score += 20
	}
	if meta.Robots != "" {
		score += 20
	}

	meta.Score = score
	return meta
}

func (a *Analyzer) analyzeHeaders(doc *goquery.Document) HeaderAnalysis {
	headers := HeaderAnalysis{}

	headers.H1Count = doc.Find("h1").Length()
	headers.H2Count = doc.Find("h2").Length()
	headers.H3Count = doc.Find("h3").Length()

	doc.Find("h1").Each(func(_ int, s *goquery.Selection) {
		headers.H1Text = append(headers.H1Text, strings.TrimSpace(s.Text()))
	})

	// Score calculation
	score := 0
	if headers.H1Count == 1 {
		score += 40
	} else if headers.H1Count > 1 {
		score += 20
	}

	if headers.H2Count > 0 {
		score += 30
	}

	if headers.H3Count > 0 {
		score += 30
	}

	headers.Score = score
	return headers
}

func (a *Analyzer) analyzeContent(doc *goquery.Document) ContentAnalysis {
	content := ContentAnalysis{
		KeywordDensity: make(map[string]float64),
	}

	// Word count
	text := doc.Find("body").Text()
	words := strings.Fields(text)
	content.WordCount = len(words)

	// Image analysis
	images := doc.Find("img")
	content.TotalImages = images.Length()
	content.HasImages = content.TotalImages > 0

	images.Each(func(_ int, s *goquery.Selection) {
		if _, exists := s.Attr("alt"); exists {
			content.ImagesWithAlt++
		}
	})

	// Calculate score
	score := 0
	if content.WordCount >= 300 {
		score += 30
	}
	if content.HasImages {
		score += 20
		if content.ImagesWithAlt == content.TotalImages {
			score += 30
		} else if content.ImagesWithAlt > 0 {
			score += 20
		}
	}

	content.Score = score
	return content
}

func (a *Analyzer) analyzePerformance(pageSize int, loadTime time.Duration, mobileOptimized bool) Performance {
	perf := Performance{
		PageSize:        pageSize,
		LoadTime:        int(loadTime.Milliseconds()),
		MobileOptimized: mobileOptimized,
		PageSizeSeverity: "good",
		LoadTimeSeverity: "good",
	}

	// Score calculation - Total 100 points possible
	score := 100

	// Page Size scoring (40 points)
	// Convert pageSize to KB for easier reading
	pageSizeKB := float64(pageSize) / 1024.0

	switch {
	case pageSizeKB > 5120: // > 5MB
		score -= 40 // Critical issue
		perf.PageSizeSeverity = "critical"
	case pageSizeKB > 2048: // > 2MB
		score -= 30 // Major issue
		perf.PageSizeSeverity = "major"
	case pageSizeKB > 1024: // > 1MB
		score -= 20 // Moderate issue
		perf.PageSizeSeverity = "moderate"
	case pageSizeKB > 500: // > 500KB
		score -= 10 // Minor issue
		perf.PageSizeSeverity = "minor"
	}

	// Load Time scoring (40 points)
	loadTimeMs := loadTime.Milliseconds()
	switch {
	case loadTimeMs > 3000: // > 3s
		score -= 40 // Critical issue
		perf.LoadTimeSeverity = "critical"
	case loadTimeMs > 2000: // > 2s
		score -= 30 // Major issue
		perf.LoadTimeSeverity = "major"
	case loadTimeMs > 1500: // > 1.5s
		score -= 20 // Moderate issue
		perf.LoadTimeSeverity = "moderate"
	case loadTimeMs > 1000: // > 1s
		score -= 10 // Minor issue
		perf.LoadTimeSeverity = "minor"
	}

	// Mobile Optimization scoring (20 points)
	if !perf.MobileOptimized {
		score -= 20
	}

	perf.Score = score
	return perf
}

// analyzeLinksWithContext analyzes links with context awareness
func (a *Analyzer) analyzeLinksWithContext(ctx context.Context, doc *goquery.Document, baseURL string) LinkAnalysis {
	links := LinkAnalysis{}
	
	// Get a map from the pool
	checkedLinks := mapPool.Get().(map[string]bool)
	for k := range checkedLinks {
		delete(checkedLinks, k)
	}
	defer mapPool.Put(checkedLinks)
	
	// Get a URL slice from the pool
	linkURLs := urlSlicePool.Get().([]string)
	linkURLs = linkURLs[:0] // Reset the slice while keeping capacity
	defer urlSlicePool.Put(linkURLs)

	// First, collect all unique links
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}

		// Clean and normalize the URL
		href = strings.TrimSpace(href)
		if strings.HasPrefix(href, "//") {
			href = "https:" + href
		} else if strings.HasPrefix(href, "/") {
			href = baseURL + href
		}

		// Skip if we've already seen this link
		if checkedLinks[href] {
			return
		}
		checkedLinks[href] = true
		
		// Categorize the link
		if strings.HasPrefix(href, baseURL) || strings.HasPrefix(href, "/") {
			links.InternalLinks++
			linkURLs = append(linkURLs, href)
		} else if strings.HasPrefix(href, "http") {
			links.ExternalLinks++
			linkURLs = append(linkURLs, href)
		}
	})
	
	// Now check all links concurrently with controlled parallelism
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit to 10 concurrent requests
	var mu sync.Mutex // Mutex to protect the brokenLinks counter
	
	// Create a context that will be canceled when the function returns
	linkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	for _, url := range linkURLs {
		// Check if the parent context is canceled
		select {
		case <-ctx.Done():
			// Parent context canceled, stop processing
			return links
		default:
			// Continue processing
		}
		
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore
			
			if !a.isLinkAccessibleWithContext(linkCtx, url) {
				mu.Lock()
				links.BrokenLinks++
				mu.Unlock()
			}
		}(url)
	}
	
	// Use a channel to signal completion or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	// Wait for completion or context cancellation
	select {
	case <-done:
		// All links checked successfully
	case <-ctx.Done():
		// Context canceled, return what we have so far
	}

	// Score calculation - Total 100 points possible
	score := 100

	// Internal Links scoring (40 points)
	switch {
	case links.InternalLinks == 0:
		score -= 40 // Critical issue
	case links.InternalLinks < 3:
		score -= 30 // Major issue
	case links.InternalLinks < 5:
		score -= 20 // Moderate issue
	}

	// External Links scoring (30 points)
	switch {
	case links.ExternalLinks == 0:
		score -= 30 // Missing external links
	case links.ExternalLinks > 50:
		score -= 15 // Too many external links
	}

	// Broken Links scoring (30 points)
	switch {
	case links.BrokenLinks > 5:
		score -= 30 // Critical issue
	case links.BrokenLinks > 3:
		score -= 20 // Major issue
	case links.BrokenLinks > 0:
		score -= 10 // Minor issue
	}

	links.Score = score
	return links
}

// isLinkAccessibleWithContext checks if a link is accessible with context support
func (a *Analyzer) isLinkAccessibleWithContext(ctx context.Context, url string) bool {
	// Check cache first
	cacheKey := generateCacheKey(url)
	a.linkCacheMutex.RLock()
	if entry, found := a.linkCache[cacheKey]; found {
		if time.Since(entry.timestamp) < a.linkCacheTTL {
			a.stats.IncrementStats(0, 0, 1, 0) // Increment link cache hits
			a.linkCacheMutex.RUnlock()
			return entry.accessible
		}
	}
	a.linkCacheMutex.RUnlock()
	
	// Not in cache or expired
	a.stats.IncrementStats(0, 0, 0, 1) // Increment link cache misses
	
	// Create a request with context
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return a.cacheAndReturnLinkStatus(cacheKey, false)
	}
	
	// Set user agent to avoid being blocked by some websites
	req.Header.Set("User-Agent", "SEOAnalyzer/1.0")
	
	// Create a client with a shorter timeout for link checking
	client := &http.Client{
		Timeout: 5 * time.Second, // Shorter timeout just for link checking
		Transport: a.client.Transport,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return a.cacheAndReturnLinkStatus(cacheKey, false)
	}
	defer resp.Body.Close()
	
	accessible := resp.StatusCode >= 200 && resp.StatusCode < 400
	return a.cacheAndReturnLinkStatus(cacheKey, accessible)
}

// cacheAndReturnLinkStatus caches the link status and returns it
func (a *Analyzer) cacheAndReturnLinkStatus(cacheKey string, accessible bool) bool {
	a.linkCacheMutex.Lock()
	defer a.linkCacheMutex.Unlock()
	
	a.linkCache[cacheKey] = linkCacheEntry{
		accessible: accessible,
		timestamp:  time.Now(),
	}
	
	return accessible
}

// For backward compatibility
func (a *Analyzer) analyzeLinks(doc *goquery.Document, baseURL string) LinkAnalysis {
	return a.analyzeLinksWithContext(context.Background(), doc, baseURL)
}

// For backward compatibility
func (a *Analyzer) isLinkAccessible(url string) bool {
	return a.isLinkAccessibleWithContext(context.Background(), url)
}

func (a *Analyzer) calculateOverallScore(analysis *SEOAnalysis) float64 {
	weights := map[string]float64{
		"title":       0.2,
		"meta":        0.2,
		"headers":     0.15,
		"content":     0.2,
		"performance": 0.15,
		"links":       0.1,
	}

	score := 0.0
	score += float64(analysis.Title.Score) * weights["title"]
	score += float64(analysis.Meta.Score) * weights["meta"]
	score += float64(analysis.Headers.Score) * weights["headers"]
	score += float64(analysis.Content.Score) * weights["content"]
	score += float64(analysis.Performance.Score) * weights["performance"]
	score += float64(analysis.Links.Score) * weights["links"]

	return score
}

func (a *Analyzer) generateRecommendations(analysis *SEOAnalysis) []string {
	var recommendations []string

	// Title recommendations
	if !analysis.Title.HasTitle {
		recommendations = append(recommendations, "Add a title tag to your page")
	} else if analysis.Title.Length < 30 {
		recommendations = append(recommendations, "Title tag is too short (should be 30-60 characters)")
	} else if analysis.Title.Length > 60 {
		recommendations = append(recommendations, "Title tag is too long (should be 30-60 characters)")
	}

	// Meta recommendations
	if !analysis.Meta.HasDescription {
		recommendations = append(recommendations, "Add a meta description")
	} else if analysis.Meta.DescriptionLen < 120 {
		recommendations = append(recommendations, "Meta description is too short (should be 120-160 characters)")
	} else if analysis.Meta.DescriptionLen > 160 {
		recommendations = append(recommendations, "Meta description is too long (should be 120-160 characters)")
	}

	// Headers recommendations
	if analysis.Headers.H1Count == 0 {
		recommendations = append(recommendations, "Add an H1 heading")
	} else if analysis.Headers.H1Count > 1 {
		recommendations = append(recommendations, "Multiple H1 headings found - consider using only one")
	}

	// Content recommendations
	if analysis.Content.WordCount < 300 {
		recommendations = append(recommendations, "Add more content (aim for at least 300 words)")
	}
	if analysis.Content.TotalImages > 0 && analysis.Content.ImagesWithAlt < analysis.Content.TotalImages {
		recommendations = append(recommendations, "Add alt text to all images")
	}

	// Performance recommendations
	pageSizeKB := float64(analysis.Performance.PageSize) / 1024.0
	if pageSizeKB > 5120 {
		recommendations = append(recommendations, 
			"Critical: Page size is extremely large (>5MB). Consider optimizing images, minifying CSS/JS, and removing unnecessary resources")
	} else if pageSizeKB > 2048 {
		recommendations = append(recommendations, 
			"Major: Page size is very large (>2MB). Optimize images and consider lazy loading for non-critical resources")
	} else if pageSizeKB > 1024 {
		recommendations = append(recommendations, 
			"Moderate: Page size is large (>1MB). Look for opportunities to optimize images and resources")
	} else if pageSizeKB > 500 {
		recommendations = append(recommendations, 
			"Minor: Page size is above optimal (>500KB). Consider basic optimization techniques")
	}

	if analysis.Performance.LoadTime > 3000 {
		recommendations = append(recommendations, 
			"Critical: Page load time is extremely slow (>3s). Consider using a CDN, optimizing server response time, and reducing resource size")
	} else if analysis.Performance.LoadTime > 2000 {
		recommendations = append(recommendations, 
			"Major: Page load time is slow (>2s). Optimize server response time and consider resource optimization")
	} else if analysis.Performance.LoadTime > 1500 {
		recommendations = append(recommendations, 
			"Moderate: Page load time is above optimal (>1.5s). Look for opportunities to improve performance")
	} else if analysis.Performance.LoadTime > 1000 {
		recommendations = append(recommendations, 
			"Minor: Page load time is slightly above optimal (>1s). Consider fine-tuning performance")
	}

	if !analysis.Performance.MobileOptimized {
		recommendations = append(recommendations, 
			"Add a proper viewport meta tag for mobile optimization (e.g., <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">)")
	}

	// Links recommendations
	if analysis.Links.BrokenLinks > 0 {
		recommendations = append(recommendations, 
			"Fix broken links: Found " + strconv.Itoa(analysis.Links.BrokenLinks) + " broken link(s)")
	}
	if analysis.Links.InternalLinks < 3 {
		recommendations = append(recommendations, 
			"Add more internal links to improve site navigation and SEO (aim for at least 3-5)")
	}
	if analysis.Links.ExternalLinks == 0 {
		recommendations = append(recommendations, 
			"Add relevant external links to authoritative sources to improve content credibility")
	} else if analysis.Links.ExternalLinks > 50 {
		recommendations = append(recommendations, 
			"Consider reducing the number of external links (current: " + strconv.Itoa(analysis.Links.ExternalLinks) + ") to maintain focus")
	}

	return recommendations
}

// GetStats returns the statistics storage instance
func (a *Analyzer) GetStats() *stats.Storage {
	return a.stats
}

// Shutdown performs cleanup and ensures all statistics are saved
func (a *Analyzer) Shutdown() error {
	if a == nil {
		return nil
	}

	// Stop the cleanup goroutine by closing a channel
	if a.stats != nil {
		if err := a.stats.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown stats storage: %w", err)
		}
	}

	// Clear caches
	a.cacheMutex.Lock()
	a.cache = nil
	a.cacheMutex.Unlock()

	a.linkCacheMutex.Lock()
	a.linkCache = nil
	a.linkCacheMutex.Unlock()

	return nil
}
