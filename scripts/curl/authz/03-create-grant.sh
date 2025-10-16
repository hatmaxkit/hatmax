#!/bin/bash

# Test authz service create grant endpoint with optional user creation
# Usage: ./03-create-grant.sh [user_id] [role_name] [resource] [create_user]
# Example: ./03-create-grant.sh user123 admin /api/users
# Example with user creation: ./03-create-grant.sh "" admin /api/users true
# If no parameters provided, creates a new user and uses defaults

# Configuration
AUTHN_URL="http://localhost:8082"
AUTHZ_URL="http://localhost:8083"
TIMESTAMP=$(date +%s)

# Use provided parameters or defaults
USER_ID=${1:-""}
ROLE_NAME=${2:-"admin"}
RESOURCE=${3:-"/api/users"}
CREATE_USER=${4:-"true"}

echo "Testing authz create grant..."

# Create user if USER_ID is empty or CREATE_USER is true
if [ -z "$USER_ID" ] || [ "$CREATE_USER" = "true" ]; then
    echo "Creating new user in AuthN..."
    TEST_EMAIL="granttest${TIMESTAMP}@example.com"
    TEST_PASSWORD="TestPassword123!"
    
    SIGNUP_RESPONSE=$(curl -s -X POST $AUTHN_URL/authn/signup \
      -H "Content-Type: application/json" \
      -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\"
      }")
    
    echo "Signup Response: $SIGNUP_RESPONSE"
    
    # Try to extract user ID
    NEW_USER_ID=$(echo $SIGNUP_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4 || echo "")
    
    if [ -n "$NEW_USER_ID" ]; then
        USER_ID=$NEW_USER_ID
        echo "✅ User created with ID: $USER_ID"
    else
        echo "⚠️  Could not extract user ID, using fallback UUID"
        USER_ID=$(uuidgen 2>/dev/null || printf "550e8400-e29b-41d4-a716-%012x" $TIMESTAMP)
    fi
fi

echo "User ID: $USER_ID"
echo "Role: $ROLE_NAME" 
echo "Resource: $RESOURCE"
echo ""

# First ensure the role exists
echo "Ensuring role '$ROLE_NAME' exists..."
ROLE_RESPONSE=$(curl -s -X POST $AUTHZ_URL/authz/roles \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"$ROLE_NAME\",
    \"description\": \"Role for grant testing\",
    \"permissions\": [\"read:users\", \"write:users\"]
  }")

echo "Role creation response: $ROLE_RESPONSE"
echo ""

# Create the grant
echo "Creating grant..."
curl -X POST $AUTHZ_URL/authz/grants \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"role_name\": \"$ROLE_NAME\",
    \"resource\": \"$RESOURCE\"
  }" \
  -w "\nHTTP Status: %{http_code}\n" \
  -v

echo -e "\n--- Test completed ---"
echo "Grant created for user: $USER_ID"
echo "You can use this USER_ID for other tests: $USER_ID"
