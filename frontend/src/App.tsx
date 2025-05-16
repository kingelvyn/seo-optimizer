import React, { useState, useMemo, useCallback, useEffect } from 'react';
import './App.css';
import LoadingSkeleton from './components/LoadingSkeleton';
import SeoSuggestions from './components/SeoSuggestions';
import ScoreIndicator from './components/ScoreIndicator';
import AnalysisProgress from './components/AnalysisProgress';
import SectionInfo from './components/SectionInfo';
import Statistics from './components/Statistics';

interface AnalysisResult {
  url: string;
  score: number;
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
  links: {
    internalLinks: number;
    externalLinks: number;
    brokenLinks: number;
    score: number;
  };
  recommendations: string[];
}

const getImageAnalysisStatus = (total: number, withAlt: number) => {
  if (total === 0) return { severity: 'critical', message: 'No images found - consider adding relevant images' };
  const percentage = (withAlt / total) * 100;
  if (percentage === 100) return { severity: 'good', message: 'All images have alt text' };
  if (percentage >= 80) return { severity: 'moderate', message: 'Most images have alt text' };
  return { severity: 'warning', message: 'Many images missing alt text - improve accessibility' };
};

const getMetaTagSeverity = (meta: any) => {
  let score = 0;
  if (meta.hasDescription) score += 30;
  if (meta.hasKeywords) score += 20;
  if (meta.viewport) score += 25;
  if (meta.robots) score += 25;

  if (score >= 90) return 'good';
  if (score >= 70) return 'moderate';
  return 'warning';
};

function App() {
  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<AnalysisResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState('init');
  const [progress, setProgress] = useState(0);

  const apiUrl = process.env.NODE_ENV === 'development' 
    ? 'http://localhost:8082/api'
    : (process.env.REACT_APP_API_URL || '/api');

  // Simulate analysis progress
  useEffect(() => {
    if (!loading) {
      setProgress(0);
      setCurrentStep('init');
      return;
    }

    const steps = ['init', 'meta', 'content', 'performance', 'links', 'final'];
    const totalSteps = steps.length;
    let currentStepIndex = 0;

    // Faster initial steps, slower for resource-intensive steps
    const stepTiming = {
      init: 300,      // Quick initialization
      meta: 300,      // Meta tags are quick to check
      content: 400,   // Content analysis takes a bit longer
      performance: 500, // Performance checks need more time
      links: 600,     // Link checking is usually the longest
      final: 300      // Quick recommendation generation
    };

    const updateProgress = () => {
      if (currentStepIndex < totalSteps) {
        const currentStepName = steps[currentStepIndex];
        setCurrentStep(currentStepName);
        
        // Calculate cumulative progress
        const completedProgress = currentStepIndex * (100 / totalSteps);
        const stepProgress = (100 / totalSteps);
        setProgress(Math.round(completedProgress + stepProgress));
        
        currentStepIndex++;
        
        // Schedule next step
        if (currentStepIndex < totalSteps) {
          setTimeout(updateProgress, stepTiming[steps[currentStepIndex] as keyof typeof stepTiming]);
        }
      }
    };

    // Start the progress animation
    updateProgress();

    return () => {
      currentStepIndex = totalSteps; // Ensure cleanup stops the animation
    };
  }, [loading]);

  const handleAnalyze = useCallback(async () => {
    if (!url) {
      setError('Please enter a URL');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setProgress(0);
      setCurrentStep('init');
      
      const startTime = Date.now();
      
      const response = await fetch(`${apiUrl}/analyze`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url }),
      });

      if (!response.ok) {
        throw new Error('Analysis failed');
      }

      const data = await response.json();
      
      // Ensure minimum animation time for UX
      const analysisTime = Date.now() - startTime;
      const minAnimationTime = 2000; // Minimum 2 seconds for smooth UX
      
      if (analysisTime < minAnimationTime) {
        await new Promise(resolve => setTimeout(resolve, minAnimationTime - analysisTime));
      }
      
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setLoading(false);
    }
  }, [url, apiUrl]);

  const formatBytes = useCallback((bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }, []);

  const formatTime = useCallback((ms: number) => {
    return ms > 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`;
  }, []);

  const getScoreColor = useCallback((score: number) => {
    if (score >= 80) return 'text-green-600';
    if (score >= 60) return 'text-yellow-600';
    return 'text-red-600';
  }, []);

  const getStatusIndicator = useCallback((value: string | boolean, type: 'severity' | 'boolean') => {
    if (type === 'boolean') {
      return (
        <span className={`status-indicator ${value ? 'status-good' : 'status-error'}`}>
          {value ? 'Yes' : 'No'}
        </span>
      );
    }

    if (type === 'severity') {
      const severityValue = value as string;
      switch (severityValue) {
        case 'critical':
          return <span className="status-indicator status-critical">Critical</span>;
        case 'major':
          return <span className="status-indicator status-critical">Major</span>;
        case 'moderate':
          return <span className="status-indicator status-warning">Moderate</span>;
        case 'minor':
          return <span className="status-indicator status-warning">Minor</span>;
        case 'good':
        default:
          return <span className="status-indicator status-good">Good</span>;
      }
    }

    return null;
  }, []);

  const renderLoadingSkeleton = useCallback(() => {
    return (
      <div className="result-container">
        <h2>Analyzing...</h2>
        <AnalysisProgress currentStep={currentStep} progress={progress} />
        <div className="skeleton-section">
          <div className="skeleton-header">
            <LoadingSkeleton lines={1} />
          </div>
          <div className="skeleton-content">
            <LoadingSkeleton lines={3} />
          </div>
        </div>
      </div>
    );
  }, [currentStep, progress]);

  // Memoize the analysis sections to prevent unnecessary re-renders
  const analysisContent = useMemo(() => {
    if (!result) return null;
    
    const imageAnalysis = result.content ? 
      getImageAnalysisStatus(result.content.totalImages, result.content.imagesWithAlt) : 
      { severity: 'critical', message: 'No images analyzed' };

    const metaSeverity = result.meta ? getMetaTagSeverity(result.meta) : 'warning';

    return (
      <div className="result-container">
        <h2>Analysis Results</h2>
        <div className="overall-score">
          <h3>Overall SEO Score</h3>
          <ScoreIndicator score={result.score} size="large" />
        </div>

        <SeoSuggestions
          title={result.title}
          meta={result.meta}
          headers={result.headers}
          content={result.content}
          performance={result.performance}
          metaSeverity={metaSeverity}
          recommendations={result.recommendations || []}
        />

        <div className="analysis-sections-grid">
          {/* Title Analysis */}
          {result.title && (
            <div className="analysis-section">
              <div className="analysis-section-header">
                <h3>
                  <span className="section-icon">T</span>
                  Title Analysis
                </h3>
                <ScoreIndicator score={result.title.score} size="small" showText={false} />
              </div>
              <div className="section-content">
                <p><strong>Title:</strong> {result.title.title || 'No title found'}</p>
                <p><strong>Length:</strong> {result.title.length} characters</p>
              </div>
              <SectionInfo type="title" />
            </div>
          )}

          {/* Meta Tags Analysis */}
          {result.meta && (
            <div className="analysis-section">
              <div className="analysis-section-header">
                <h3>
                  <span className="section-icon">M</span>
                  Meta Tags
                </h3>
                <ScoreIndicator score={result.meta.score} size="small" showText={false} />
              </div>
              <div className="section-content">
                <p><strong>Description:</strong> {result.meta.description || 'Not set'}</p>
                <p><strong>Description Length:</strong> {result.meta.descriptionLength} characters</p>
                <p><strong>Keywords:</strong> {result.meta.keywords || 'Not set'}</p>
                <p><strong>Robots:</strong> {result.meta.robots || 'Not set'}</p>
                <p><strong>Viewport:</strong> {result.meta.viewport || 'Not set'}</p>
              </div>
              <SectionInfo type="meta" />
            </div>
          )}

          {/* Headers Analysis */}
          {result.headers && (
            <div className="analysis-section">
              <div className="analysis-section-header">
                <h3>
                  <span className="section-icon">H</span>
                  Headers Structure
                </h3>
                <ScoreIndicator score={result.headers.score} size="small" showText={false} />
              </div>
              <div className="section-content">
                <p><strong>H1 Tags:</strong> {result.headers.h1Count}</p>
                <p><strong>H2 Tags:</strong> {result.headers.h2Count}</p>
                <p><strong>H3 Tags:</strong> {result.headers.h3Count}</p>
                {result.headers.h1Text && result.headers.h1Text.length > 0 && (
                  <div>
                    <strong>H1 Content:</strong>
                    <ul>
                      {result.headers.h1Text.map((text, index) => (
                        <li key={index}>{text}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
              <SectionInfo type="headers" />
            </div>
          )}

          {/* Content Analysis */}
          {result.content && (
            <div className="analysis-section">
              <div className="analysis-section-header">
                <h3>
                  <span className="section-icon">C</span>
                  Content Analysis
                </h3>
                <ScoreIndicator score={result.content.score} size="small" showText={false} />
              </div>
              <div className="section-content">
                <p>
                  <strong>Word Count:</strong> 
                  {result.content.wordCount}
                  {result.content.wordCount < 300 ? 
                    getStatusIndicator('moderate', 'severity') : 
                    getStatusIndicator('good', 'severity')}
                </p>
                <p>
                  <strong>Images:</strong> 
                  {result.content.totalImages} 
                  {result.content.totalImages > 0 && ` (${Math.round((result.content.imagesWithAlt / result.content.totalImages) * 100)}% with alt text)`}
                  {getStatusIndicator(imageAnalysis.severity, 'severity')}
                </p>
                <p className="analysis-detail">
                  <span className="detail-icon">ℹ️</span>
                  {imageAnalysis.message}
                </p>
              </div>
              <SectionInfo type="content" />
            </div>
          )}

          {/* Performance Metrics */}
          {result.performance && (
            <div className="analysis-section">
              <div className="analysis-section-header">
                <h3>
                  <span className="section-icon">P</span>
                  Performance
                </h3>
                <ScoreIndicator score={result.performance.score} size="small" showText={false} />
              </div>
              <div className="section-content">
                <p>
                  <strong>Load Time:</strong> 
                  {formatTime(result.performance.loadTime)}
                  {getStatusIndicator(result.performance.loadTimeSeverity, 'severity')}
                </p>
                <p>
                  <strong>Page Size:</strong> 
                  {formatBytes(result.performance.pageSize)}
                  {getStatusIndicator(result.performance.pageSizeSeverity, 'severity')}
                </p>
                <p>
                  <strong>Mobile Optimized:</strong>
                  {getStatusIndicator(result.performance.mobileOptimized, 'boolean')}
                </p>
              </div>
              <SectionInfo type="performance" />
            </div>
          )}

          {/* Links Analysis */}
          {result.links && (
            <div className="analysis-section">
              <div className="analysis-section-header">
                <h3>
                  <span className="section-icon">L</span>
                  Links Analysis
                </h3>
                <ScoreIndicator score={result.links.score} size="small" showText={false} />
              </div>
              <div className="section-content">
                <p><strong>Internal Links:</strong> {result.links.internalLinks}</p>
                <p><strong>External Links:</strong> {result.links.externalLinks}</p>
                <p>
                  <strong>Broken Links:</strong> 
                  {result.links.brokenLinks}
                  {result.links.brokenLinks > 0 ? 
                    getStatusIndicator('critical', 'severity') : 
                    getStatusIndicator('good', 'severity')}
                </p>
              </div>
              <SectionInfo type="links" />
            </div>
          )}
        </div>
      </div>
    );
  }, [result, formatBytes, formatTime, getStatusIndicator]);

  return (
    <div className="App">
      <header className="App-header">
        <h1>SEO Optimizer</h1>
        <div className="analysis-form">
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="Enter website URL"
            className="url-input"
          />
          <button
            onClick={handleAnalyze}
            disabled={loading}
            className="analyze-button"
          >
            {loading ? 'Analyzing...' : 'Analyze'}
          </button>
        </div>

        {error && <div className="error-message">{error}</div>}
        {loading ? renderLoadingSkeleton() : analysisContent}
        
        <Statistics apiUrl={apiUrl} />
      </header>
      <footer className="footer">
        <span>
          <span className="copyright-symbol">&copy;</span>
          Elvynprise 2025
        </span>
      </footer>
    </div>
  );
}

export default App;
