import React from 'react';

interface ScoreIndicatorProps {
  score: number;
  size?: 'small' | 'medium' | 'large';
  showText?: boolean;
}

const ScoreIndicator: React.FC<ScoreIndicatorProps> = ({ 
  score, 
  size = 'medium',
  showText = true 
}) => {
  const getScoreColor = (score: number) => {
    if (score >= 75) return '#059669'; // green
    if (score >= 50) return '#d97706'; // yellow
    return '#dc2626'; // red
  };

  const getScoreText = (score: number) => {
    if (score >= 75) return 'Good';
    if (score >= 50) return 'Needs\nImprovement';
    return 'Poor';
  };

  const formatScore = (score: number) => {
    // If score is a whole number, don't show decimal places
    return Number.isInteger(score) ? score.toString() : score.toFixed(1);
  };

  const getSizeClass = (size: string) => {
    switch (size) {
      case 'small':
        return 'score-indicator-small';
      case 'large':
        return 'score-indicator-large';
      default:
        return 'score-indicator-medium';
    }
  };

  const color = getScoreColor(score);
  const progress = `${score}%`;

  return (
    <div className={`score-indicator ${getSizeClass(size)}`}>
      <div 
        className="score-circle"
        style={{ 
          '--progress-color': color,
          '--progress': progress
        } as React.CSSProperties}
      >
        <div className="score-inner">
          <span className="score-value" style={{ color }}>
            {formatScore(score)}%
          </span>
          {showText && (
            <span 
              className="score-label" 
              style={{ color }}
              title={getScoreText(score).replace('\n', ' ')}
            >
              {getScoreText(score)}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

export default ScoreIndicator; 