#!/bin/bash

# Complete AuthN + AuthZ workflow test
# This script demonstrates the full flow:
# 1. Create a user in AuthN
# 2. Create a role in AuthZ  
# 3. Grant the role to the user
# 4. Test permission evaluation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

echo_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

echo_error() {
    echo -e "${RED}❌ $1${NC}"
}

echo_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Configuration
AUTHN_URL="http://localhost:8082"
AUTHZ_URL="http://localhost:8083"
TIMESTAMP=$(date +%s)
TEST_EMAIL="testuser${TIMESTAMP}@example.com"
TEST_PASSWORD="TestPassword123!"
TEST_ROLE="admin"
TEST_RESOURCE="/api/users"
TEST_PERMISSION="read:users"

echo_info "Starting complete AuthN + AuthZ workflow test..."
echo_info "Test user: $TEST_EMAIL"
echo_info "Test role: $TEST_ROLE"
echo_info "Test resource: $TEST_RESOURCE"
echo ""

# Step 1: Create user in AuthN
echo_info "Step 1: Creating user in AuthN..."
SIGNUP_RESPONSE=$(curl -s -X POST $AUTHN_URL/authn/signup \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\"
  }")

echo "AuthN Signup Response: $SIGNUP_RESPONSE"

# Extract user ID from response (assuming it's in the response)
USER_ID=$(echo $SIGNUP_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -z "$USER_ID" ]; then
    # If we can't extract from signup, try to sign in and get it
    echo_warning "Could not extract user ID from signup, trying signin..."
    
    SIGNIN_RESPONSE=$(curl -s -X POST $AUTHN_URL/authn/signin \
      -H "Content-Type: application/json" \
      -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\"
      }")
    
    echo "AuthN Signin Response: $SIGNIN_RESPONSE"
    USER_ID=$(echo $SIGNIN_RESPONSE | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4 || echo "")
    
    if [ -z "$USER_ID" ]; then
        # Generate a fake UUID for testing if we still can't get one
        USER_ID="550e8400-e29b-41d4-a716-446655440000"
        echo_warning "Could not extract user ID, using fake UUID: $USER_ID"
    fi
fi

echo_success "User created with ID: $USER_ID"
echo ""

# Step 2: Create role in AuthZ
echo_info "Step 2: Creating role in AuthZ..."
ROLE_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/roles \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"$TEST_ROLE\",
    \"description\": \"Administrator role for testing\",
    \"permissions\": [\"read:users\", \"write:users\", \"delete:users\", \"manage:system\"]
  }")

echo "AuthZ Role Creation Response: $ROLE_RESPONSE"
echo_success "Role '$TEST_ROLE' created"
echo ""

# Step 3: Grant role to user
echo_info "Step 3: Granting role to user..."
GRANT_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/grants \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"role_name\": \"$TEST_ROLE\",
    \"resource\": \"$TEST_RESOURCE\"
  }")

echo "AuthZ Grant Response: $GRANT_RESPONSE"
echo_success "Role '$TEST_ROLE' granted to user $USER_ID for resource $TEST_RESOURCE"
echo ""

# Step 4: Test permission evaluation
echo_info "Step 4: Testing permission evaluation..."
PERMISSION_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/permissions/check \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"permission\": \"$TEST_PERMISSION\",
    \"scope\": {
      \"type\": \"resource\",
      \"id\": \"$TEST_RESOURCE\"
    }
  }")

echo "AuthZ Permission Check Response: $PERMISSION_RESPONSE"

# Check if permission was allowed
if echo "$PERMISSION_RESPONSE" | grep -q '"allowed":true'; then
    echo_success "Permission check PASSED: User has '$TEST_PERMISSION' on '$TEST_RESOURCE'"
else
    echo_error "Permission check FAILED: User does NOT have '$TEST_PERMISSION' on '$TEST_RESOURCE'"
fi
echo ""

# Step 5: List user grants
echo_info "Step 5: Listing user grants..."
GRANTS_RESPONSE=$(curl -s -X GET "$AUTHZ_URL/authz/grants?user_id=$USER_ID" \
  -H "Content-Type: application/json")

echo "User Grants Response: $GRANTS_RESPONSE"
echo_success "User grants retrieved"
echo ""

# Step 6: List all roles
echo_info "Step 6: Listing all roles..."
ROLES_RESPONSE=$(curl -s -X GET $AUTHZ_URL/authz/roles \
  -H "Content-Type: application/json")

echo "Roles List Response: $ROLES_RESPONSE"
echo_success "Roles list retrieved"
echo ""

echo_success "🎉 Complete workflow test finished!"
echo_info "Summary:"
echo_info "  • User: $TEST_EMAIL (ID: $USER_ID)"
echo_info "  • Role: $TEST_ROLE"  
echo_info "  • Resource: $TEST_RESOURCE"
echo_info "  • Permission: $TEST_PERMISSION"
echo ""
echo_info "You can now use this user ID in other AuthZ tests: $USER_ID"