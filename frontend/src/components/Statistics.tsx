import React, { useEffect, useState } from 'react';

interface Statistics {
  uniqueVisitors24h: number;
  totalRequests: number;
  errorRate: number;
  averageLoadTime: number;
  popularUrls: { [key: string]: number };
}

interface StatisticsProps {
  apiUrl: string;
}

const Statistics: React.FC<StatisticsProps> = ({ apiUrl }) => {
  const [stats, setStats] = useState<Statistics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch(`${apiUrl}/statistics`);
        if (!response.ok) {
          throw new Error('Failed to fetch statistics');
        }
        const data = await response.json();
        setStats(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load statistics');
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 60000); // Update every minute

    return () => clearInterval(interval);
  }, [apiUrl]);

  if (loading) {
    return (
      <div className="statistics-panel loading">
        <div className="skeleton-line"></div>
        <div className="skeleton-line"></div>
        <div className="skeleton-line"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="statistics-panel error">
        <p>Error loading statistics: {error}</p>
      </div>
    );
  }

  if (!stats) {
    return null;
  }

  return (
    <div className="statistics-panel">
      <h3>
        <span className="section-icon">S</span>
        Statistics Dashboard
      </h3>
      <div className="stats-content">
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-title">Unique Visitors (24h)</div>
            <div className="stat-value">{stats.uniqueVisitors24h}</div>
          </div>
          <div className="stat-card">
            <div className="stat-title">Total Analyses</div>
            <div className="stat-value">{stats.totalRequests}</div>
          </div>
          <div className="stat-card">
            <div className="stat-title">Error Rate</div>
            <div className="stat-value">{stats.errorRate.toFixed(1)}%</div>
          </div>
          <div className="stat-card">
            <div className="stat-title">Avg. Load Time</div>
            <div className="stat-value">{stats.averageLoadTime.toFixed(0)}ms</div>
          </div>
        </div>
        
        <div className="popular-urls">
          <h4>Most Analyzed URLs</h4>
          <ul>
            {Object.entries(stats.popularUrls)
              .sort(([, a], [, b]) => b - a)
              .slice(0, 5)
              .map(([url, count]) => (
                <li key={url}>
                  <span className="url-text">{url}</span>
                  <span className="url-count">{count}</span>
                </li>
              ))}
          </ul>
        </div>
      </div>
    </div>
  );
};

export default Statistics; 