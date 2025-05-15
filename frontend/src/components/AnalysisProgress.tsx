import React from 'react';

interface AnalysisProgressProps {
  currentStep: string;
  progress: number;
}

const analysisSteps = [
  { id: 'init', label: 'Initializing Analysis' },
  { id: 'meta', label: 'Checking Meta Information' },
  { id: 'content', label: 'Analyzing Content' },
  { id: 'performance', label: 'Measuring Performance' },
  { id: 'links', label: 'Checking Links' },
  { id: 'final', label: 'Generating Recommendations' }
];

const AnalysisProgress: React.FC<AnalysisProgressProps> = ({ currentStep, progress }) => {
  return (
    <div className="analysis-progress">
      <div className="progress-bar-container">
        <div 
          className="progress-bar" 
          style={{ width: `${progress}%` }}
        >
          <span className="progress-text">{progress}%</span>
        </div>
      </div>
      <div className="analysis-steps">
        {analysisSteps.map((step) => (
          <div 
            key={step.id}
            className={`step ${currentStep === step.id ? 'active' : ''} ${
              analysisSteps.findIndex(s => s.id === step.id) < 
              analysisSteps.findIndex(s => s.id === currentStep) ? 'completed' : ''
            }`}
          >
            <div className="step-indicator"></div>
            <span className="step-label">{step.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default AnalysisProgress; 