#!/bin/bash

# Test authz service check permission endpoint with full setup
# Usage: ./04-check-permission.sh [user_id] [permission] [resource] [setup]
# Example: ./04-check-permission.sh user123 read:users /api/users
# Example with full setup: ./04-check-permission.sh "" read:users /api/users true
# If setup=true, creates user, role, and grant before checking permission

# Configuration
AUTHN_URL="http://localhost:8082"
AUTHZ_URL="http://localhost:8083"
TIMESTAMP=$(date +%s)

# Use provided parameters or defaults
USER_ID=${1:-""}
PERMISSION=${2:-"read:users"}
RESOURCE=${3:-"/api/users"}
SETUP=${4:-"true"}

echo "Testing authz check permission..."

# If setup is true and no USER_ID provided, create complete setup
if [ "$SETUP" = "true" ] && [ -z "$USER_ID" ]; then
    echo "🏗️  Setting up complete test scenario..."
    TEST_EMAIL="permtest${TIMESTAMP}@example.com"
    TEST_PASSWORD="TestPassword123!"
    TEST_ROLE="reader"
    
    # Step 1: Create user
    echo "1️⃣  Creating user in AuthN..."
    SIGNUP_RESPONSE=$(curl -s -X POST $AUTHN_URL/authn/signup \
      -H "Content-Type: application/json" \
      -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\"
      }")
    
    USER_ID=$(echo $SIGNUP_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4 || echo "")
    
    if [ -z "$USER_ID" ]; then
        # Generate a proper UUID v4
        USER_ID=$(uuidgen 2>/dev/null || printf "550e8400-e29b-41d4-a716-%012x" $TIMESTAMP)
        echo "⚠️  Using fallback UUID: $USER_ID"
    else
        echo "✅ User created: $TEST_EMAIL (ID: $USER_ID)"
    fi
    
    # Step 2: Create role
    echo "2️⃣  Creating role in AuthZ..."
    curl -s -X POST $AUTHZ_URL/authz/roles \
      -H "Content-Type: application/json" \
      -d "{
        \"name\": \"$TEST_ROLE\",
        \"description\": \"Reader role for permission testing\",
        \"permissions\": [\"$PERMISSION\"]
      }" > /dev/null
    
    echo "✅ Role '$TEST_ROLE' created with permission '$PERMISSION'"
    
    # Step 3: Grant role to user  
    echo "3️⃣  Granting role to user..."
    curl -s -X POST $AUTHZ_URL/authz/grants \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"$USER_ID\",
        \"role_name\": \"$TEST_ROLE\",
        \"resource\": \"$RESOURCE\"
      }" > /dev/null
    
    echo "✅ Role '$TEST_ROLE' granted to user for resource '$RESOURCE'"
    echo ""
fi

# Default USER_ID if still empty
if [ -z "$USER_ID" ]; then
    # Generate a proper UUID v4 for testing
    USER_ID=$(uuidgen 2>/dev/null || printf "550e8400-e29b-41d4-a716-%012x" $TIMESTAMP)
fi

echo "🔍 Testing permission check..."
echo "User ID: $USER_ID"
echo "Permission: $PERMISSION"
echo "Resource: $RESOURCE"
echo ""

# Check permission
PERMISSION_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/policy/evaluate \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"permission\": \"$PERMISSION\",
    \"scope\": {
      \"type\": \"resource\",
      \"id\": \"$RESOURCE\"
    }
  }")

echo "Permission Check Response:"
echo "$PERMISSION_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$PERMISSION_RESPONSE"

# Check result
if echo "$PERMISSION_RESPONSE" | grep -q '"allowed":true'; then
    echo ""
    echo "✅ SUCCESS: User has '$PERMISSION' permission on '$RESOURCE'"
else
    echo ""
    echo "❌ DENIED: User does NOT have '$PERMISSION' permission on '$RESOURCE'"
fi

echo ""
echo "--- Test completed ---"
echo "User ID for future tests: $USER_ID"
