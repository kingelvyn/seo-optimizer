# SEO Optimizer

A modern web application for analyzing and optimizing website SEO performance. Built with Go (backend) and React/TypeScript (frontend).

## Features

- Real-time SEO analysis
- Comprehensive metrics including:
  - Title and meta tags optimization
  - Header structure analysis
  - Content quality assessment
  - Performance metrics (load time, page size)
  - Mobile optimization check
  - Internal and external link analysis
- Visual score indicators
- Detailed recommendations for improvement
- Modern, responsive UI

## Tech Stack

### Backend
- Go 1.x
- Gin web framework
- goquery for HTML parsing

### Frontend
- React 18
- TypeScript
- Modern CSS

## Getting Started

### Prerequisites
- Go 1.x
- Node.js 16+
- npm or yarn

### Installation

1. Clone the repository
```bash
git clone https://github.com/yourusername/seo-optimizer.git
cd seo-optimizer
```

2. Install backend dependencies
```bash
cd backend
go mod download
```

3. Install frontend dependencies
```bash
cd frontend
npm install
```

### Running the Application

1. Start the backend server
```bash
cd backend
go run main.go
```

2. Start the frontend development server
```bash
cd frontend
npm start
```

The application will be available at `http://localhost:3000`

## API Endpoints

### POST /api/analyze
Analyzes a website's SEO performance

Request body:
```json
{
  "url": "https://example.com"
}
```

Response:
```json
{
  "url": "https://example.com",
  "score": 85,
  "title": { ... },
  "meta": { ... },
  "headers": { ... },
  "content": { ... },
  "performance": { ... },
  "links": { ... },
  "recommendations": [ ... ]
}
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [goquery](https://github.com/PuerkitoBio/goquery) for HTML parsing
- React and TypeScript communities
- All contributors to this project 