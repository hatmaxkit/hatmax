#!/bin/bash

# Complete AuthN + AuthZ Integration Test
# This script tests the full workflow: user creation, role creation, grants, and permission checking
# Usage: ./06-complete-integration-test.sh [email] [role_name] [permission] [resource]

# Configuration
AUTHN_URL="http://localhost:8082"
AUTHZ_URL="http://localhost:8083"
TIMESTAMP=$(date +%s)

# Use provided parameters or defaults
TEST_EMAIL=${1:-"integration${TIMESTAMP}@example.com"}
ROLE_NAME=${2:-"integration_tester"}
PERMISSION=${3:-"read:todos"}
RESOURCE=${4:-"/api/todos"}
TEST_PASSWORD="TestPassword123!"

echo "🚀 Starting Complete AuthN + AuthZ Integration Test"
echo "==============================================="
echo "📧 Email: $TEST_EMAIL"
echo "👤 Role: $ROLE_NAME" 
echo "🔑 Permission: $PERMISSION"
echo "📦 Resource: $RESOURCE"
echo ""

# Step 1: Create user in AuthN
echo "1️⃣  Creating user in AuthN..."
SIGNUP_RESPONSE=$(curl -s -X POST $AUTHN_URL/authn/signup \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\"
  }")

echo "📋 Signup Response: $SIGNUP_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$SIGNUP_RESPONSE"

# Extract user ID
USER_ID=$(echo $SIGNUP_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$USER_ID" ]; then
    echo "❌ FAILED: Could not create user or extract user ID"
    echo "Response: $SIGNUP_RESPONSE"
    exit 1
fi

echo "✅ User created successfully!"
echo "   📧 Email: $TEST_EMAIL"
echo "   🆔 User ID: $USER_ID"
echo ""

# Step 2: Create role in AuthZ
echo "2️⃣  Creating role '$ROLE_NAME' in AuthZ..."
ROLE_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/roles \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"$ROLE_NAME\",
    \"description\": \"Integration test role for $PERMISSION\",
    \"permissions\": [\"$PERMISSION\"]
  }")

echo "📋 Role Response: $ROLE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$ROLE_RESPONSE"

if echo "$ROLE_RESPONSE" | grep -q '"error"'; then
    echo "⚠️  Role creation had issues, but continuing (role might already exist)"
else
    echo "✅ Role '$ROLE_NAME' created successfully with permission '$PERMISSION'"
fi
echo ""

# Step 3: Grant role to user
echo "3️⃣  Granting role '$ROLE_NAME' to user for resource '$RESOURCE'..."
GRANT_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/grants \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"role_name\": \"$ROLE_NAME\",
    \"resource\": \"$RESOURCE\"
  }")

echo "📋 Grant Response: $GRANT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$GRANT_RESPONSE"

if echo "$GRANT_RESPONSE" | grep -q '"error"'; then
    echo "❌ FAILED: Could not create grant"
    echo "Response: $GRANT_RESPONSE"
    exit 1
fi

echo "✅ Grant created successfully!"
echo "   👤 User: $USER_ID"
echo "   🎭 Role: $ROLE_NAME"
echo "   📦 Resource: $RESOURCE"
echo ""

# Step 4: Test permission evaluation
echo "4️⃣  Testing permission evaluation..."
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

echo "📋 Permission Check Response:"
echo "$PERMISSION_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$PERMISSION_RESPONSE"
echo ""

# Evaluate result
if echo "$PERMISSION_RESPONSE" | grep -q '"allowed":true'; then
    echo "🎉 SUCCESS! Complete integration test PASSED!"
    echo "   ✅ User creation: OK"
    echo "   ✅ Role creation: OK" 
    echo "   ✅ Grant assignment: OK"
    echo "   ✅ Permission evaluation: ALLOWED"
else
    echo "❌ FAILED! Permission was denied"
    echo "   ✅ User creation: OK"
    echo "   ✅ Role creation: OK"
    echo "   ✅ Grant assignment: OK"
    echo "   ❌ Permission evaluation: DENIED"
    exit 1
fi

echo ""
echo "==============================================="
echo "🎯 Integration Test Summary"
echo "   📧 User Email: $TEST_EMAIL"
echo "   🆔 User ID: $USER_ID"
echo "   🎭 Role: $ROLE_NAME"
echo "   🔑 Permission: $PERMISSION"
echo "   📦 Resource: $RESOURCE"
echo "   ✅ Result: PERMISSION GRANTED"
echo "==============================================="