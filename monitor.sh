#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 SEO Optimizer Health Check"
echo "============================"
echo "Time: $(date)"
echo ""

# Check container status
echo -e "${YELLOW}Container Status:${NC}"
if docker ps --filter "name=seo-optimizer" --format "table {{.Names}}\t{{.Status}}" | grep -q "Up"; then
    echo -e "${GREEN}✓ Containers are running${NC}"
else
    echo -e "${RED}✗ One or more containers are not running${NC}"
fi

# Check recent errors in logs
echo -e "\n${YELLOW}Recent Errors in Logs:${NC}"
echo "Backend logs:"
docker logs seo-optimizer-backend-1 --tail 20 2>&1 | grep -i "error\|fail\|warn" || echo "No errors found in recent logs"

# Test API endpoints
echo -e "\n${YELLOW}API Health Checks:${NC}"

# Health check
echo "Testing /api/health..."
if curl -s http://localhost:8082/health | grep -q "ok"; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
fi

# Cache status
echo "Testing /api/cache-status..."
if curl -s http://localhost:8082/api/cache-status > /dev/null; then
    echo -e "${GREEN}✓ Cache status endpoint responding${NC}"
else
    echo -e "${RED}✗ Cache status endpoint failed${NC}"
fi

# Statistics
echo "Testing /api/statistics..."
if curl -s http://localhost:8082/api/statistics > /dev/null; then
    echo -e "${GREEN}✓ Statistics endpoint responding${NC}"
else
    echo -e "${RED}✗ Statistics endpoint failed${NC}"
fi

# Test actual analysis endpoint
echo -e "\n${YELLOW}Testing Analysis Functionality:${NC}"
echo "Testing /api/analyze with a simple URL..."
if curl -s -X POST -H "Content-Type: application/json" \
    -d '{"url":"https://example.com"}' \
    http://localhost:8082/api/analyze > /dev/null; then
    echo -e "${GREEN}✓ Analysis endpoint responding${NC}"
else
    echo -e "${RED}✗ Analysis endpoint failed${NC}"
fi

echo -e "\n${YELLOW}Check completed at $(date)${NC}" 