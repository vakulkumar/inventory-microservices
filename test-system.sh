#!/bin/bash

# Test Script for Microservices Inventory System
# This script demonstrates how to test the system once Docker is installed

set -e

echo "=================================="
echo "Microservices System Test Script"
echo "=================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo "${BLUE}Step 1: Starting all services with Docker Compose${NC}"
docker compose up --build -d

echo ""
echo "${BLUE}Step 2: Waiting for services to be healthy...${NC}"
sleep 30

echo ""
echo "${GREEN}Step 3: Checking service health${NC}"
curl -s http://localhost:8080/health | jq '.'
curl -s http://localhost:8081/health | jq '.'
curl -s http://localhost:8082/health | jq '.'
curl -s http://localhost:8083/health | jq '.'

echo ""
echo "${GREEN}Step 4: Creating test products${NC}"
echo "Creating Laptop..."
PRODUCT1=$(curl -s -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 999.99,
    "stock": 50
  }')
echo $PRODUCT1 | jq '.'

echo ""
echo "Creating Mouse..."
PRODUCT2=$(curl -s -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wireless Mouse",
    "description": "Ergonomic wireless mouse",
    "price": 29.99,
    "stock": 100
  }')
echo $PRODUCT2 | jq '.'

echo ""
echo "Creating Keyboard..."
PRODUCT3=$(curl -s -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mechanical Keyboard",
    "description": "RGB mechanical keyboard",
    "price": 149.99,
    "stock": 8
  }')
echo $PRODUCT3 | jq '.'

echo ""
echo "${GREEN}Step 5: Listing all products${NC}"
curl -s http://localhost:8080/api/products | jq '.'

echo ""
echo "${GREEN}Step 6: Creating orders${NC}"
echo "Ordering 5 laptops..."
ORDER1=$(curl -s -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "quantity": 5
  }')
echo $ORDER1 | jq '.'

echo ""
echo "Ordering 10 mice..."
ORDER2=$(curl -s -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 2,
    "quantity": 10
  }')
echo $ORDER2 | jq '.'

echo ""
echo "Ordering 3 keyboards (should trigger low stock alert)..."
ORDER3=$(curl -s -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 3,
    "quantity": 3
  }')
echo $ORDER3 | jq '.'

echo ""
echo "${GREEN}Step 7: Listing all orders${NC}"
curl -s http://localhost:8080/api/orders | jq '.'

echo ""
echo "${GREEN}Step 8: Checking updated stock levels${NC}"
curl -s http://localhost:8080/api/products | jq '.[] | {id, name, stock}'

echo ""
echo "${BLUE}Step 9: Checking Kafka events in Notification Service${NC}"
echo "Check notification service logs:"
docker compose logs notification-service | tail -20

echo ""
echo "${BLUE}Step 10: Accessing Prometheus Metrics${NC}"
echo "Sample metrics from Inventory Service:"
curl -s http://localhost:8081/metrics | grep "inventory_http_requests_total\|inventory_stock_levels" | head -10

echo ""
echo "${GREEN}Step 11: Load Testing${NC}"
echo "Generating 100 requests..."
for i in {1..100}; do
  curl -s http://localhost:8080/api/products > /dev/null
  if [ $((i % 20)) -eq 0 ]; then
    echo "Completed $i requests..."
  fi
done

echo ""
echo "${GREEN}=================================="
echo "Testing Complete!"
echo "=================================="
echo ""
echo "Access the following UIs:"
echo "  - API Gateway:  http://localhost:8080"
echo "  - Prometheus:   http://localhost:9090"
echo "  - Grafana:      http://localhost:3000 (admin/admin)"
echo ""
echo "Try these Prometheus queries:"
echo "  - rate(inventory_http_requests_total[1m])"
echo "  - histogram_quantile(0.95, sum(rate(order_http_request_duration_seconds_bucket[1m])) by (le))"
echo "  - inventory_stock_levels"
echo ""
echo "To stop all services:"
echo "  docker compose down"
echo ""
