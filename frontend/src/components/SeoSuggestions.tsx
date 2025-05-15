import React from 'react';

interface SeoSuggestionsProps {
  title: {
    title: string;
    length: number;
    hasTitle: boolean;
    score: number;
  };
  meta: {
    description: string;
    descriptionLength: number;
    hasDescription: boolean;
    keywords: string;
    hasKeywords: boolean;
    robots: string;
    viewport: string;
    score: number;
  };
  headers: {
    h1Count: number;
    h2Count: number;
    h3Count: number;
    h1Text: string[];
    score: number;
  };
  content: {
    wordCount: number;
    keywordDensity: { [key: string]: number };
    hasImages: boolean;
    imagesWithAlt: number;
    totalImages: number;
    score: number;
  };
  performance: {
    pageSize: number;
    loadTime: number;
    mobileOptimized: boolean;
    score: number;
    pageSizeSeverity: string;
    loadTimeSeverity: string;
  };
  metaSeverity: string;
  recommendations: string[];
}

const SeoSuggestions: React.FC<SeoSuggestionsProps> = ({
  title,
  meta,
  headers,
  content,
  performance,
  metaSeverity,
  recommendations
}) => {
  const getSuggestions = () => {
    const suggestions: { tip: string; priority: 'high' | 'medium' | 'low'; category: string }[] = [];

    // Title suggestions
    if (!title.hasTitle) {
      suggestions.push({
        tip: 'Add a title tag to your page - it\'s crucial for SEO',
        priority: 'high',
        category: 'Title',
      });
    } else if (title.length < 30) {
      suggestions.push({
        tip: 'Your title is too short. Aim for 50-60 characters for optimal visibility in search results',
        priority: 'medium',
        category: 'Title',
      });
    } else if (title.length > 60) {
      suggestions.push({
        tip: 'Your title is too long. Keep it under 60 characters to prevent truncation in search results',
        priority: 'medium',
        category: 'Title',
      });
    }

    // Meta suggestions
    if (!meta.hasDescription) {
      suggestions.push({
        tip: 'Add a meta description to improve click-through rates from search results',
        priority: 'high',
        category: 'Meta Tags',
      });
    } else if (meta.descriptionLength < 120) {
      suggestions.push({
        tip: 'Your meta description is too short. Aim for 120-155 characters',
        priority: 'medium',
        category: 'Meta Tags',
      });
    } else if (meta.descriptionLength > 155) {
      suggestions.push({
        tip: 'Your meta description is too long. Keep it under 155 characters',
        priority: 'medium',
        category: 'Meta Tags',
      });
    }

    if (!meta.hasKeywords) {
      suggestions.push({
        tip: 'Add meta keywords to help search engines understand your content focus',
        priority: 'medium',
        category: 'Meta Tags',
      });
    }

    if (!meta.robots) {
      suggestions.push({
        tip: 'Consider adding a robots meta tag to control search engine crawling',
        priority: 'medium',
        category: 'Meta Tags',
      });
    }

    if (!meta.viewport) {
      suggestions.push({
        tip: 'Add a viewport meta tag for better mobile optimization',
        priority: 'high',
        category: 'Meta Tags',
      });
    }

    // Headers suggestions
    if (headers.h1Count === 0) {
      suggestions.push({
        tip: 'Add an H1 heading to your page - every page should have exactly one H1',
        priority: 'high',
        category: 'Headers',
      });
    } else if (headers.h1Count > 1) {
      suggestions.push({
        tip: 'Multiple H1 headings detected. Use only one H1 per page',
        priority: 'medium',
        category: 'Headers',
      });
    }

    if (headers.h2Count === 0) {
      suggestions.push({
        tip: 'Add H2 headings to structure your content better',
        priority: 'medium',
        category: 'Headers',
      });
    }

    // Content suggestions
    if (content.wordCount < 300) {
      suggestions.push({
        tip: 'Your content is too thin. Aim for at least 300 words for better rankings',
        priority: 'high',
        category: 'Content',
      });
    }

    if (!content.hasImages) {
      suggestions.push({
        tip: 'Add relevant images to make your content more engaging',
        priority: 'medium',
        category: 'Content',
      });
    } else if (content.imagesWithAlt < content.totalImages) {
      suggestions.push({
        tip: `Add alt text to ${content.totalImages - content.imagesWithAlt} images for better accessibility and SEO`,
        priority: 'medium',
        category: 'Content',
      });
    }

    // Performance suggestions
    const formatSize = (size: number) => {
      const kb = size / 1024;
      if (kb > 1024) {
        return `${(kb / 1024).toFixed(2)}MB`;
      }
      return `${kb.toFixed(2)}KB`;
    };

    const formatTime = (ms: number) => {
      return ms > 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`;
    };

    // Page size suggestions
    if (performance.pageSizeSeverity !== 'good') {
      suggestions.push({
        tip: `Page size (${formatSize(performance.pageSize)}) needs optimization. ${
          performance.pageSizeSeverity === 'critical' ? 'Critical: Immediate attention needed.' :
          performance.pageSizeSeverity === 'major' ? 'Major: Consider significant optimizations.' :
          performance.pageSizeSeverity === 'moderate' ? 'Moderate: Look for optimization opportunities.' :
          'Minor: Consider basic optimizations.'
        }`,
        priority: performance.pageSizeSeverity === 'critical' || performance.pageSizeSeverity === 'major' ? 'high' : 'medium',
        category: 'Performance',
      });
    }

    // Load time suggestions
    if (performance.loadTimeSeverity !== 'good') {
      suggestions.push({
        tip: `Load time (${formatTime(performance.loadTime)}) needs improvement. ${
          performance.loadTimeSeverity === 'critical' ? 'Critical: Page is loading very slowly.' :
          performance.loadTimeSeverity === 'major' ? 'Major: Page load time needs significant improvement.' :
          performance.loadTimeSeverity === 'moderate' ? 'Moderate: Consider performance optimizations.' :
          'Minor: Page load time could be improved.'
        }`,
        priority: performance.loadTimeSeverity === 'critical' || performance.loadTimeSeverity === 'major' ? 'high' : 'medium',
        category: 'Performance',
      });
    }

    if (!performance.mobileOptimized) {
      suggestions.push({
        tip: 'Your page is not mobile-friendly. Implement responsive design',
        priority: 'high',
        category: 'Performance',
      });
    }

    return suggestions;
  };

  const suggestions = getSuggestions();

  return (
    <div className="seo-suggestions">
      <h3>
        <span className="section-icon">O</span>
        Optimization Suggestions
      </h3>
      <div className="suggestions-container">
        {suggestions.map((suggestion, index) => (
          <div key={index} className={`suggestion-card ${suggestion.priority}`}>
            <div className="suggestion-category">{suggestion.category}</div>
            <div className="suggestion-content">
              <div className="priority-badge">{suggestion.priority}</div>
              <p>{suggestion.tip}</p>
            </div>
          </div>
        ))}
      </div>
      
      <div className="recommendations">
        <h3>
          <span className="section-icon">R</span>
          Recommendations
        </h3>
        <ul>
          {recommendations.map((recommendation: string, index: number) => (
            <li key={index}>{recommendation}</li>
          ))}
        </ul>
      </div>
    </div>
  );
};

export default SeoSuggestions; 