# SEO Optimizer Health Check Baseline
Date: 2025-06-13 16:57:50

## Container Status
Both containers are running normally:
- Frontend: `seo-optimizer-frontend-1` (Up 30 minutes)
- Backend: `seo-optimizer-backend-1` (Up 30 minutes)

## Log Analysis
No errors, warnings, or failures found in recent logs. Normal operation logs show:
- Regular stats saving (every minute)
- Visitor tracking
- Single unique visitor (IP: 192.168.1.104)

## API Endpoint Status

### Cache Status
```json
{
  "isCached": false,
  "stats": {
    "analysisEntries": 2,
    "linkEntries": 0,
    "analysisCacheHits": 0,
    "linkCacheHits": 0,
    "analysisCacheMisses": 2,
    "linkCacheMisses": 17,
    "analysisCacheTTL": 1800000000000,
    "linkCacheTTL": 600000000000
  },
  "url": ""
}
```

### Statistics
```json
{
  "averageLoadTime": 1291.5,
  "errorRate": 0,
  "totalRequests": 2,
  "uniqueVisitors24h": 2
}
```

### Analysis Endpoint
Successfully analyzed example.com with expected response structure.

## Resource Usage
- Backend:
  - CPU: 0.00%
  - Memory: 10.71MiB / 8GiB (0.13%)
  - Network I/O: 216kB / 68.3kB
  - Block I/O: 3.22MB / 889kB
  - PIDs: 9

- Frontend:
  - CPU: 0.00%
  - Memory: 59.78MiB / 8GiB (0.73%)
  - Network I/O: 218kB / 12.7kB
  - Block I/O: 37.6MB / 4.1kB
  - PIDs: 11

## Key Metrics to Monitor
1. Memory usage (both containers well under 1%)
2. Error rate (currently 0%)
3. Cache statistics (2 analysis entries, 17 link cache misses)
4. Average load time (1291.5ms)
5. Number of unique visitors (2 in 24h)

## Notes
- All endpoints responding normally
- No errors in logs
- Resource usage is very low
- Cache is functioning as expected
- Analysis endpoint successfully processing requests 