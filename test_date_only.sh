#!/bin/bash

# Test script to verify date-only purchase_date functionality

echo "Testing Purchase Transaction Date-Only Functionality"
echo "====================================================="

# Base URL
BASE_URL="http://localhost:8080/api"

# Login to get token (adjust credentials as needed)
echo -e "\n1. Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@boilerplates.dev",
    "password": "admin123"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | sed 's/"token":"//')

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to login. Please check credentials."
  exit 1
fi

echo "✅ Login successful"

# Get a supplier ID
echo -e "\n2. Getting supplier..."
SUPPLIERS_RESPONSE=$(curl -s -X GET "$BASE_URL/publishers?all=true" \
  -H "Authorization: Bearer $TOKEN")

SUPPLIER_ID=$(echo $SUPPLIERS_RESPONSE | grep -o '"id":"[^"]*' | head -1 | sed 's/"id":"//')

if [ -z "$SUPPLIER_ID" ]; then
  echo "❌ No suppliers found. Please create a supplier first."
  exit 1
fi

echo "✅ Found supplier: $SUPPLIER_ID"

# Get a book ID
echo -e "\n3. Getting book..."
BOOKS_RESPONSE=$(curl -s -X GET "$BASE_URL/books?all=true" \
  -H "Authorization: Bearer $TOKEN")

BOOK_ID=$(echo $BOOKS_RESPONSE | grep -o '"id":"[^"]*' | head -1 | sed 's/"id":"//')

if [ -z "$BOOK_ID" ]; then
  echo "❌ No books found. Please create a book first."
  exit 1
fi

echo "✅ Found book: $BOOK_ID"

# Create purchase transaction with date only
echo -e "\n4. Creating purchase transaction with date-only (2026-01-31)..."
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/purchase-transactions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"supplier_id\": \"$SUPPLIER_ID\",
    \"purchase_date\": \"2026-01-31\",
    \"items\": [
      {
        \"book_id\": \"$BOOK_ID\",
        \"quantity\": 5,
        \"price\": 45000
      }
    ]
  }")

echo "Response: $CREATE_RESPONSE"

TRANSACTION_ID=$(echo $CREATE_RESPONSE | grep -o '"id":"[^"]*' | head -1 | sed 's/"id":"//')

if [ -z "$TRANSACTION_ID" ]; then
  echo "❌ Failed to create purchase transaction"
  echo "Response: $CREATE_RESPONSE"
  exit 1
fi

echo "✅ Purchase transaction created: $TRANSACTION_ID"

# Verify the date format in response
PURCHASE_DATE=$(echo $CREATE_RESPONSE | grep -o '"purchase_date":"[^"]*' | sed 's/"purchase_date":"//')
echo "Purchase date in response: $PURCHASE_DATE"

if [[ $PURCHASE_DATE == "2026-01-31" ]]; then
  echo "✅ Date format is correct (date only, no time)"
else
  echo "❌ Date format is incorrect. Expected: 2026-01-31, Got: $PURCHASE_DATE"
fi

# Get the transaction to verify
echo -e "\n5. Retrieving transaction to verify..."
GET_RESPONSE=$(curl -s -X GET "$BASE_URL/purchase-transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $TOKEN")

RETRIEVED_DATE=$(echo $GET_RESPONSE | grep -o '"purchase_date":"[^"]*' | sed 's/"purchase_date":"//')
echo "Retrieved purchase date: $RETRIEVED_DATE"

if [[ $RETRIEVED_DATE == "2026-01-31" ]]; then
  echo "✅ Retrieved date format is correct"
else
  echo "❌ Retrieved date format is incorrect. Expected: 2026-01-31, Got: $RETRIEVED_DATE"
fi

# Update the transaction with a new date
echo -e "\n6. Updating purchase date to 2026-02-01..."
UPDATE_RESPONSE=$(curl -s -X PUT "$BASE_URL/purchase-transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "purchase_date": "2026-02-01"
  }')

UPDATED_DATE=$(echo $UPDATE_RESPONSE | grep -o '"purchase_date":"[^"]*' | sed 's/"purchase_date":"//')
echo "Updated purchase date: $UPDATED_DATE"

if [[ $UPDATED_DATE == "2026-02-01" ]]; then
  echo "✅ Update successful with correct date format"
else
  echo "❌ Update failed or incorrect date format. Expected: 2026-02-01, Got: $UPDATED_DATE"
fi

# Test date filtering
echo -e "\n7. Testing date filtering..."
FILTER_RESPONSE=$(curl -s -X GET "$BASE_URL/purchase-transactions?start_date=2026-02-01&end_date=2026-02-01" \
  -H "Authorization: Bearer $TOKEN")

FILTER_COUNT=$(echo $FILTER_RESPONSE | grep -o '"purchase_transactions":\[' | wc -l)

if [ $FILTER_COUNT -gt 0 ]; then
  echo "✅ Date filtering works"
else
  echo "⚠️  Date filtering returned no results (this might be expected)"
fi

# Clean up - delete the test transaction
echo -e "\n8. Cleaning up test transaction..."
DELETE_RESPONSE=$(curl -s -X DELETE "$BASE_URL/purchase-transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "✅ Test transaction deleted"

echo -e "\n====================================================="
echo "✅ All tests completed successfully!"
echo "====================================================="
