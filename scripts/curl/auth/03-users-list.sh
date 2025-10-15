#!/bin/bash

# Test auth service users list endpoint
# Usage: ./03-users-list.sh

echo "Testing auth users list..."

curl -X GET http://localhost:8081/users \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -v

echo -e "\n--- Test completed ---"