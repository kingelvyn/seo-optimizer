import React from 'react';

interface LoadingSkeletonProps {
  lines?: number;
}

const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({ lines = 3 }) => {
  return (
    <div className="skeleton-wrapper">
      {Array.from({ length: lines }).map((_, index) => (
        <div key={index} className="skeleton-line">
          <div className="skeleton-animation"></div>
        </div>
      ))}
    </div>
  );
};

export default LoadingSkeleton; 