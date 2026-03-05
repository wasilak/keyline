#!/bin/bash

set -e

echo "🚀 Keyline Testing Script"
echo "========================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if keyline is running
if ! curl -s http://localhost:9000/healthz > /dev/null 2>&1; then
    echo -e "${RED}❌ Keyline is not running on port 9000${NC}"
    echo "Start it with: ./bin/keyline --config config/test-config.yaml"
    exit 1
fi

echo -e "${GREEN}✅ Keyline is running${NC}"
echo ""

# Test 1: Health Check
echo -e "${BLUE}Test 1: Health Check${NC}"
curl -s http://localhost:9000/healthz | jq .
echo ""

# Test 2: Metrics Endpoint
echo -e "${BLUE}Test 2: Metrics Endpoint${NC}"
curl -s http://localhost:9000/metrics | head -20
echo "... (truncated)"
echo ""

# Test 3: Basic Auth - Valid Credentials
echo -e "${BLUE}Test 3: Basic Auth - Valid Credentials${NC}"
RESPONSE=$(curl -s -u testuser:password http://localhost:9000/get)
echo "$RESPONSE" | jq .
echo ""

# Check if ES header was injected
if echo "$RESPONSE" | jq -e '.headers."X-Es-Authorization"' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ X-Es-Authorization header injected${NC}"
else
    echo -e "${RED}❌ X-Es-Authorization header missing${NC}"
fi
echo ""

# Test 4: Basic Auth - Invalid Credentials
echo -e "${BLUE}Test 4: Basic Auth - Invalid Credentials${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -u testuser:wrongpassword http://localhost:9000/get)
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✅ Correctly returned 401 Unauthorized${NC}"
else
    echo -e "${RED}❌ Expected 401, got $HTTP_CODE${NC}"
fi
echo ""

# Test 5: No Auth
echo -e "${BLUE}Test 5: No Authentication${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9000/get)
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✅ Correctly returned 401 Unauthorized${NC}"
else
    echo -e "${RED}❌ Expected 401, got $HTTP_CODE${NC}"
fi
echo ""

# Test 6: Different HTTP Methods
echo -e "${BLUE}Test 6: POST Request with Basic Auth${NC}"
curl -s -u testuser:password -X POST http://localhost:9000/post -d '{"test":"data"}' -H "Content-Type: application/json" | jq .
echo ""

echo -e "${GREEN}🎉 All tests completed!${NC}"
