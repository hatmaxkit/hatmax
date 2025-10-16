#!/bin/bash

# Test authz service list roles endpoint
# Usage: ./02-list-roles.sh

echo "Testing authz list roles..."

curl -X GET http://localhost:8083/authz/roles \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -v

echo -e "\n--- Test completed ---"