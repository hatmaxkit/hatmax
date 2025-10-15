#!/bin/bash

# Full auth service test flow
# Usage: ./test-flow.sh [email]
# If no email provided, generates a unique one for the flow

set -e  # Exit on any error

# Generate unique email for this test run
if [ "$1" != "" ]; then
    TEST_EMAIL="$1"
else
    TIMESTAMP=$(date +%s)
    TEST_EMAIL="flow${TIMESTAMP}@example.com"
fi

echo "🚀 Starting full auth service test flow..."
echo "📧 Using email: $TEST_EMAIL"
echo

# Test 1: Signup
echo "📝 Test 1: Signup"
./01-signup.sh "$TEST_EMAIL"
echo

# Wait a bit
sleep 1

# Test 2: Signin
echo "🔐 Test 2: Signin"
./02-signin.sh "$TEST_EMAIL"
echo

# Wait a bit  
sleep 1

# Test 3: List users
echo "👥 Test 3: List users"
./03-users-list.sh
echo

echo "✅ All tests completed with email: $TEST_EMAIL"
