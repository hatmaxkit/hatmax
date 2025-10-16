#!/bin/bash

# Test authz service create role endpoint
# Usage: ./01-create-role.sh [role_name]
# Example: ./01-create-role.sh admin
# If no role name provided, generates a unique one

# Use provided role name or generate unique one
if [ "$1" != "" ]; then
    ROLE_NAME="$1"
else
    # Generate unique role name with timestamp
    TIMESTAMP=$(date +%s)
    ROLE_NAME="role_${TIMESTAMP}"
fi

echo "Testing authz create role with name: $ROLE_NAME"

curl -X POST http://localhost:8083/authz/roles \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"$ROLE_NAME\",
    \"description\": \"Test role created via script\",
    \"permissions\": [\"read:users\", \"write:users\"]
  }" \
  -w "\nHTTP Status: %{http_code}\n" \
  -v

echo -e "\n--- Test completed ---"
echo "Role name used: $ROLE_NAME"