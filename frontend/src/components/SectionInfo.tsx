import React from 'react';

interface SectionInfoProps {
  type: 'title' | 'meta' | 'headers' | 'content' | 'performance' | 'links';
}

const sectionInfo: Record<SectionInfoProps['type'], { title: string; description: string; metrics: string[] }> = {
  title: {
    title: 'Title Tag Analysis',
    description: 'The title tag is one of the most important SEO elements. It appears in search results and browser tabs.',
    metrics: [
      'Length: Optimal title length is 50-60 characters',
      'Presence: Every page should have a unique title tag',
      'Relevance: Title should contain main keywords and accurately describe the page'
    ]
  },
  meta: {
    title: 'Meta Tags Information',
    description: 'Meta tags provide search engines with information about your page content and structure.',
    metrics: [
      'Description: Should be 150-160 characters and summarize page content',
      'Keywords: While less important now, they should reflect page topics',
      'Robots: Controls search engine crawling and indexing',
      'Viewport: Essential for mobile responsiveness'
    ]
  },
  headers: {
    title: 'Headers Structure',
    description: 'Headers (H1-H6) create a hierarchical structure for your content, helping both users and search engines understand your page organization.',
    metrics: [
      'H1: Should be unique and appear only once per page',
      'H2-H3: Use for subsections and maintain proper hierarchy',
      'Content: Headers should be descriptive and contain relevant keywords'
    ]
  },
  content: {
    title: 'Content Analysis',
    description: 'Content quality and structure are crucial for both SEO and user experience.',
    metrics: [
      'Word Count: Longer content (>300 words) typically ranks better',
      'Image Alt Text: Helps accessibility and SEO',
      'Keyword Density: Should be natural, typically 1-3%'
    ]
  },
  performance: {
    title: 'Performance Metrics',
    description: 'Page performance directly impacts user experience and search engine rankings.',
    metrics: [
      'Load Time: Should be under 3 seconds for optimal experience',
      'Page Size: Smaller pages load faster (ideal < 2MB)',
      'Mobile Optimization: Essential for modern SEO'
    ]
  },
  links: {
    title: 'Links Analysis',
    description: 'Links help search engines discover and understand relationships between pages.',
    metrics: [
      'Internal Links: Help website navigation and spread link equity',
      'External Links: Add credibility and reference sources',
      'Broken Links: Should be fixed to maintain user experience and SEO value'
    ]
  }
};

const SectionInfo: React.FC<SectionInfoProps> = ({ type }) => {
  const info = sectionInfo[type];

  return (
    <div className="section-info">
      <h4>{info.title}</h4>
      <p>{info.description}</p>
      <ul>
        {info.metrics.map((metric, index) => (
          <li key={index}>{metric}</li>
        ))}
      </ul>
    </div>
  );
};

export default SectionInfo; 