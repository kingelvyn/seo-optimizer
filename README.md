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
- Statistics Dashboard:
  - Monthly statistics tracking with persistence
  - Automatic data retention management
  - Unique visitors tracking
  - Analysis request monitoring
  - Error rate tracking
  - Average load time metrics
  - Most analyzed URLs (development mode)
  - Cache performance metrics
  - Graceful shutdown with data preservation
- Environment-aware configuration
- Persistent statistics storage with automatic cleanup

## Tech Stack

### Backend
- Go 1.21
- Gin web framework
- goquery for HTML parsing
- Custom statistics tracking with:
  - File-based persistence
  - Atomic writes
  - Monthly data rotation
  - Graceful shutdown handling
  - Automatic data migration
  - Buffer-based write optimization
- Automatic monthly data rotation

### Frontend
- React 18
- TypeScript
- Modern CSS
- Real-time statistics updates

### Infrastructure
- Docker
- Docker Compose
- Nginx
- Traefik (optional)
- Persistent volumes for data storage

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.21 (for local development)
- Node.js 16+ (for local development)
- npm or yarn (for local development)

### Docker Deployment

1. Clone the repository
```bash
git clone https://github.com/yourusername/seo-optimizer.git
cd seo-optimizer
```

2. Create environment files
```bash
cp backend/.env.template backend/.env
```

3. Build and run with Docker Compose
```bash
docker-compose up --build -d
```

The application will be available at `http://localhost:3001`

### Local Development

1. Install backend dependencies
```bash
cd backend
go mod download
```

2. Install frontend dependencies
```bash
cd frontend
npm install
```

3. Create local data directory
```bash
cd backend
mkdir -p data
```

4. Start the backend server (development mode)
```bash
cd backend
DEV_MODE=true go run main.go
```

5. Start the frontend development server
```bash
cd frontend
npm start
```

The application will be available at `http://localhost:3000`

## Data Persistence

### Statistics Storage
The application maintains persistent statistics across container restarts and redeployments:

- Data Directory:
  - Production: `/app/data`
  - Development: `./data` (backend/data)
- File Structure:
  - `stats.json`: Current statistics data
  - `stats.json.bak`: Backup of migrated data
- Data Retention:
  - Keeps current month and previous month
  - Automatic cleanup at midnight
  - Configurable retention period
- Write Optimization:
  - Buffer-based write system
  - Atomic file operations
  - Immediate write on shutdown
- Migration Support:
  - Automatic migration from old format
  - Data preservation during upgrades

### Graceful Shutdown
The application implements graceful shutdown to ensure data persistence:

- Catches system signals (SIGTERM/Interrupt)
- Completes pending requests (30s timeout)
- Saves statistics before exit
- Cleans up resources
- Logs shutdown process

To properly stop the application:
```bash
# Using Docker Compose
docker-compose down

# Or send SIGTERM to the container
docker stop seo-optimizer-backend-1
```

## API Endpoints

### GET /api/statistics
Retrieves current statistics (environment-aware)

Response (Development):
```json
{
  "uniqueVisitors24h": 100,
  "totalRequests": 500,
  "errorRate": 1.2,
  "averageLoadTime": 250,
  "popularUrls": {
    "https://example.com": 50,
    "https://another.com": 30
  }
}
```

Response (Production):
```json
{
  "uniqueVisitors24h": 100,
  "totalRequests": 500,
  "errorRate": 1.2,
  "averageLoadTime": 250
}
```

### GET /api/cache-status
Retrieves cache statistics and status

Response:
```json
{
  "stats": {
    "analysisEntries": 100,
    "linkEntries": 500,
    "analysisCacheHits": 80,
    "linkCacheHits": 400,
    "analysisCacheMisses": 20,
    "linkCacheMisses": 100
  },
  "url": "https://example.com",
  "isCached": true
}
```

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

## Configuration

### Environment Variables

Backend:
- `DEV_MODE`: Enable/disable development features (default: false)
- `PORT`: Server port (default: 8082)
- `GIN_MODE`: Gin framework mode (default: release)
- `DATA_DIR`: Statistics storage directory (default: /app/data in production, ./data in development)

Frontend:
- `REACT_APP_API_URL`: Backend API URL (default: /api)

## Data Persistence

The application uses a robust file-based persistence system for statistics:

### Storage Format
- JSON-based storage
- Monthly statistics segregation
- Automatic data cleanup
- Migration support for format changes

### Write Optimization
- Buffered writes for performance
- Atomic file operations
- Immediate write on shutdown
- Automatic retry mechanism

### Data Retention
- Keeps current and previous month
- Automatic cleanup at midnight
- Configurable retention period
- Safe data migration

### Deployment Considerations
- Use Docker volumes for persistence
- Proper shutdown handling
- Data directory permissions
- Environment-specific paths

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