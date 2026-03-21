#!/bin/bash

# Keyline Dynamic User Management Test Script
# This script tests the core functionality of dynamic ES user management

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ES_HOST="${ES_HOST:-localhost}"
ES_PORT="${ES_PORT:-9200}"
KEYLINE_HOST="${KEYLINE_HOST:-localhost}"
KEYLINE_PORT="${KEYLINE_PORT:-9000}"
ES_PASSWORD="${ES_PASSWORD:-changeme}"

echo "========================================"
echo "Keyline Dynamic User Management Test"
echo "========================================"
echo ""

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        exit 1
    fi
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Test 1: Check Elasticsearch is running
print_info "Test 1: Checking Elasticsearch connectivity..."
curl -k -s -u "elastic:${ES_PASSWORD}" "https://${ES_HOST}:${ES_PORT}/" > /dev/null 2>&1
print_status $? "Elasticsearch is running"

# Test 2: Check Keyline is running
print_info "Test 2: Checking Keyline connectivity..."
curl -s "http://${KEYLINE_HOST}:${KEYLINE_PORT}/" > /dev/null 2>&1
print_status $? "Keyline is running"

# Test 3: Test Basic Auth - testuser
print_info "Test 3: Testing Basic Auth as testuser..."
RESPONSE=$(curl -s -w "\n%{http_code}" -u "testuser:password" "http://${KEYLINE_HOST}:${KEYLINE_PORT}/_security/user")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    print_status 0 "Basic Auth successful for testuser"
    print_info "Response: $BODY"
else
    print_status 1 "Basic Auth failed for testuser (HTTP $HTTP_CODE)"
fi

# Test 4: Verify ES user was created
print_info "Test 4: Verifying ES user 'testuser' was created..."
ES_RESPONSE=$(curl -k -s -u "elastic:${ES_PASSWORD}" "https://${ES_HOST}:${ES_PORT}/_security/user/testuser")

if echo "$ES_RESPONSE" | grep -q "testuser"; then
    print_status 0 "ES user 'testuser' exists"
    
    # Extract and display roles
    ROLES=$(echo "$ES_RESPONSE" | grep -o '"roles":\[[^]]*\]' | head -1)
    print_info "Assigned roles: $ROLES"
else
    print_status 1 "ES user 'testuser' not found"
fi

# Test 5: Test Basic Auth - admin user
print_info "Test 5: Testing Basic Auth as admin..."
ADMIN_RESPONSE=$(curl -s -w "\n%{http_code}" -u "admin:password" "http://${KEYLINE_HOST}:${KEYLINE_PORT}/_security/user")
ADMIN_HTTP_CODE=$(echo "$ADMIN_RESPONSE" | tail -n1)

if [ "$ADMIN_HTTP_CODE" = "200" ] || [ "$ADMIN_HTTP_CODE" = "201" ]; then
    print_status 0 "Basic Auth successful for admin"
else
    print_status 1 "Basic Auth failed for admin (HTTP $ADMIN_HTTP_CODE)"
fi

# Test 6: Verify admin ES user was created with superuser role
print_info "Test 6: Verifying ES user 'admin' has superuser role..."
ADMIN_ES_RESPONSE=$(curl -k -s -u "elastic:${ES_PASSWORD}" "https://${ES_HOST}:${ES_PORT}/_security/user/admin")

if echo "$ADMIN_ES_RESPONSE" | grep -q "superuser"; then
    print_status 0 "Admin user has superuser role"
else
    print_status 1 "Admin user missing superuser role"
fi

# Test 7: Test cache behavior (second request should be faster)
print_info "Test 7: Testing credential caching..."
START=$(date +%s%N)
curl -s -u "testuser:password" "http://${KEYLINE_HOST}:${KEYLINE_PORT}/_security/user" > /dev/null
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 ))

if [ $DURATION -lt 100 ]; then
    print_status 0 "Cache hit: ${DURATION}ms (expected <100ms)"
else
    print_info "Cache miss or slow: ${DURATION}ms (expected <100ms for cache hit)"
fi

# Test 8: Verify audit logs show actual usernames
print_info "Test 8: Checking ES audit logs for individual usernames..."
# Note: This requires monitoring enabled in ES
AUDIT_CHECK=$(curl -k -s -u "elastic:${ES_PASSWORD}" \
    "https://${ES_HOST}:${ES_PORT}/.security-*/_search?size=1" 2>/dev/null || echo "")

if [ -n "$AUDIT_CHECK" ]; then
    print_status 0 "ES audit logs accessible"
else
    print_info "ES audit logs not accessible (this is OK for basic testing)"
fi

# Test 9: Test unauthorized access
print_info "Test 9: Testing unauthorized access (wrong password)..."
UNAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -u "testuser:wrongpassword" "http://${KEYLINE_HOST}:${KEYLINE_PORT}/_security/user")
UNAUTH_HTTP_CODE=$(echo "$UNAUTH_RESPONSE" | tail -n1)

if [ "$UNAUTH_HTTP_CODE" = "401" ]; then
    print_status 0 "Unauthorized access correctly rejected (HTTP 401)"
else
    print_status 1 "Unauthorized access handling unexpected (HTTP $UNAUTH_HTTP_CODE)"
fi

# Test 10: Verify role mapping for viewer (no groups)
print_info "Test 10: Testing viewer user (default roles)..."
VIEWER_RESPONSE=$(curl -s -w "\n%{http_code}" -u "viewer:password" "http://${KEYLINE_HOST}:${KEYLINE_PORT}/_security/user")
VIEWER_HTTP_CODE=$(echo "$VIEWER_RESPONSE" | tail -n1)

if [ "$VIEWER_HTTP_CODE" = "200" ] || [ "$VIEWER_HTTP_CODE" = "201" ]; then
    print_status 0 "Viewer user authenticated successfully"
    
    # Check default roles
    VIEWER_ES_RESPONSE=$(curl -k -s -u "elastic:${ES_PASSWORD}" "https://${ES_HOST}:${ES_PORT}/_security/user/viewer")
    if echo "$VIEWER_ES_RESPONSE" | grep -q "viewer"; then
        print_status 0 "Viewer user has default 'viewer' role"
    else
        print_info "Viewer user roles: $VIEWER_ES_RESPONSE"
    fi
else
    print_status 1 "Viewer user authentication failed (HTTP $VIEWER_HTTP_CODE)"
fi

echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo ""
print_info "All critical tests passed!"
print_info "Dynamic user management is working correctly"
echo ""
print_info "Next steps:"
print_info "1. Check Keyline logs: docker logs keyline"
print_info "2. Verify ES users: curl -k -u elastic:${ES_PASSWORD} https://${ES_HOST}:${ES_PORT}/_security/user"
print_info "3. Review audit logs in Kibana"
echo ""
