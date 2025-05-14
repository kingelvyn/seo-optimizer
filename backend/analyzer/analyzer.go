package analyzer

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Analyzer performs SEO analysis on a given URL
type Analyzer struct {
	client *http.Client
}

// New creates a new Analyzer instance
func New() *Analyzer {
	return &Analyzer{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Analyze performs a complete SEO analysis of the given URL
func (a *Analyzer) Analyze(url string) (*SEOAnalysis, error) {
	startTime := time.Now()

	// Fetch the page
	resp, err := a.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Calculate load time before any processing
	loadTime := time.Since(startTime)

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Check mobile optimization
	mobileOptimized := false
	doc.Find("meta[name='viewport']").Each(func(_ int, s *goquery.Selection) {
		content, exists := s.Attr("content")
		if exists && strings.Contains(strings.ToLower(content), "width=device-width") {
			mobileOptimized = true
		}
	})

	// Get actual page size from response headers if available
	pageSize := len(body)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.Atoi(contentLength); err == nil {
			pageSize = size
		}
	}

	// Perform analysis
	analysis := &SEOAnalysis{
		URL:         url,
		Title:       a.analyzeTitleTag(doc),
		Meta:        a.analyzeMetaTags(doc),
		Headers:     a.analyzeHeaders(doc),
		Content:     a.analyzeContent(doc),
		Performance: a.analyzePerformance(pageSize, loadTime, mobileOptimized),
		Links:       a.analyzeLinks(doc, url),
	}

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

func (a *Analyzer) analyzeLinks(doc *goquery.Document, baseURL string) LinkAnalysis {
	links := LinkAnalysis{}
	checkedLinks := make(map[string]bool)

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

		// Skip if we've already checked this link
		if checkedLinks[href] {
			return
		}
		checkedLinks[href] = true

		// Categorize the link
		if strings.HasPrefix(href, baseURL) || strings.HasPrefix(href, "/") {
			links.InternalLinks++
			// Check if internal link is broken
			if !a.isLinkAccessible(href) {
				links.BrokenLinks++
			}
		} else if strings.HasPrefix(href, "http") {
			links.ExternalLinks++
			// Optionally check external links (might want to limit this for performance)
			if !a.isLinkAccessible(href) {
				links.BrokenLinks++
			}
		}
	})

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

// Helper function to check if a link is accessible
func (a *Analyzer) isLinkAccessible(url string) bool {
	resp, err := a.client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
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
			"Fix broken links: Found " + string(analysis.Links.BrokenLinks) + " broken link(s)")
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
			"Consider reducing the number of external links (current: " + string(analysis.Links.ExternalLinks) + ") to maintain focus")
	}

	return recommendations
}
