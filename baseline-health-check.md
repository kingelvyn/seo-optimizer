# SEO Optimizer Health Check Baseline
Date: 2025-06-13 17:21:19

## Container Status
Both containers are running normally:
- Frontend: `seo-optimizer-frontend-1` (Up 30 minutes)
- Backend: `seo-optimizer-backend-1` (Up 30 minutes)

## Health Check Endpoint
The `/api/health` endpoint provides detailed system health information:
```json
{
  "cache": {
    "analysisCacheHits": 0,
    "analysisCacheMisses": 5,
    "analysisEntries": 0,
    "linkCacheHits": 0,
    "linkCacheMisses": 18,
    "linkEntries": 0
  },
  "memory": {
    "alloc": 1568824,
    "numGC": 1,
    "sys": 12491792,
    "totalAlloc": 3011568
  },
  "stats": {
    "errorRate": 0,
    "totalRequests": 5,
    "uniqueVisitors24h": 3
  },
  "status": "ok",
  "timestamp": "2025-06-13T17:21:19Z"
}
```

## Log Analysis
No errors, warnings, or failures found in recent logs. Normal operation logs show:
- Regular stats saving (every minute)
- Visitor tracking
- Three unique visitors (IPs: 192.168.1.104, 172.0.0.1, and one additional)

## API Endpoint Status

### Cache Status
```json
{
  "isCached": false,
  "stats": {
    "analysisEntries": 0,
    "linkEntries": 0,
    "analysisCacheHits": 0,
    "linkCacheHits": 0,
    "analysisCacheMisses": 5,
    "linkCacheMisses": 18,
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
  "totalRequests": 5,
  "uniqueVisitors24h": 3
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
3. Cache statistics (0 analysis entries, 18 link cache misses)
4. Average load time (1291.5ms)
5. Number of unique visitors (3 in 24h)
6. Health check status (currently "ok")

## Notes
- All endpoints responding normally
- No errors in logs
- Resource usage is very low
- Cache is functioning as expected
- Analysis endpoint successfully processing requests
- Health check endpoint providing detailed system status
- Memory allocation is stable (1.5MB current, 3MB total)
- Garbage collection running normally (1 collection) 