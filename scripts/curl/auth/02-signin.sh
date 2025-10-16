#!/bin/bash

# Test auth service signin endpoint
# Usage: ./02-signin.sh [email]
# Example: ./02-signin.sh user@example.com
# If no email provided, uses default test email

# Default password for all tests
PASSWORD="TestPassword123!"

# Use provided email or default
if [ "$1" != "" ]; then
    EMAIL="$1"
else
    EMAIL="test@example.com"
fi

echo "Testing auth signin with email: $EMAIL"

curl -X POST http://localhost:8082/authn/signin \
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
