#!/bin/bash

# Test auth service signup endpoint
# Usage: ./01-signup.sh [email]
# Example: ./01-signup.sh user@example.com
# If no email provided, generates a unique one

# Default password for all tests
PASSWORD="TestPassword123!"

# Use provided email or generate unique one
if [ "$1" != "" ]; then
    EMAIL="$1"
else
    # Generate unique email with timestamp
    TIMESTAMP=$(date +%s)
    EMAIL="user${TIMESTAMP}@example.com"
fi

echo "Testing auth signup with email: $EMAIL"

curl -X POST http://localhost:8082/authn/signup \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
  }" \
  -w "\nHTTP Status: %{http_code}\n" \
  -v

echo -e "\n--- Test completed ---"
echo "Email used: $EMAIL"
echo "Password used: $PASSWORD"
