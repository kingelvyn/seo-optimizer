package analyzer

// SEOAnalysis represents the complete analysis of a webpage
type SEOAnalysis struct {
	URL           string         `json:"url"`
	Title         TitleAnalysis  `json:"title"`
	Meta          MetaAnalysis   `json:"meta"`
	Headers       HeaderAnalysis `json:"headers"`
	Content       ContentAnalysis `json:"content"`
	Performance   Performance    `json:"performance"`
	Links         LinkAnalysis   `json:"links"`
	Score         float64       `json:"score"`
	Recommendations []string     `json:"recommendations"`
}

type TitleAnalysis struct {
	Title    string `json:"title"`
	Length   int    `json:"length"`
	HasTitle bool   `json:"hasTitle"`
	Score    int    `json:"score"`
}

type MetaAnalysis struct {
	Description     string `json:"description"`
	DescriptionLen  int    `json:"descriptionLength"`
	HasDescription  bool   `json:"hasDescription"`
	Keywords        string `json:"keywords"`
	HasKeywords     bool   `json:"hasKeywords"`
	Robots          string `json:"robots"`
	Viewport        string `json:"viewport"`
	Score           int    `json:"score"`
}

type HeaderAnalysis struct {
	H1Count int      `json:"h1Count"`
	H2Count int      `json:"h2Count"`
	H3Count int      `json:"h3Count"`
	H1Text  []string `json:"h1Text"`
	Score   int      `json:"score"`
}

type ContentAnalysis struct {
	WordCount        int               `json:"wordCount"`
	KeywordDensity   map[string]float64 `json:"keywordDensity"`
	HasImages        bool              `json:"hasImages"`
	ImagesWithAlt    int               `json:"imagesWithAlt"`
	TotalImages      int               `json:"totalImages"`
	Score            int               `json:"score"`
}

type Performance struct {
	PageSize        int    `json:"pageSize"`
	LoadTime        int    `json:"loadTime"`
	MobileOptimized bool   `json:"mobileOptimized"`
	Score           int    `json:"score"`
	PageSizeSeverity string `json:"pageSizeSeverity"`
	LoadTimeSeverity string `json:"loadTimeSeverity"`
}

type LinkAnalysis struct {
	InternalLinks int    `json:"internalLinks"`
	ExternalLinks int    `json:"externalLinks"`
	BrokenLinks   int    `json:"brokenLinks"`
	Score         int    `json:"score"`
} 