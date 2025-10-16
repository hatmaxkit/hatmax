#!/bin/bash

# Test authz service list user grants endpoint
# Usage: ./05-list-user-grants.sh [user_id]
# Example: ./05-list-user-grants.sh user123
# If no user_id provided, uses default

# Use provided user_id or default
USER_ID=${1:-"test_user_123"}

echo "Testing authz list grants for user: $USER_ID"

curl -X GET "http://localhost:8083/authz/grants?user_id=$USER_ID" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -v

echo -e "\n--- Test completed ---"
echo "Listed grants for user: $USER_ID"