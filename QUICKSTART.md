# Quick Start Guide

## Prerequisites Installation

```bash
# Install Docker Desktop for Mac
brew install --cask docker

# Start Docker Desktop (via GUI)
# Then verify:
docker --version
docker compose version
```

## Running the System

### Option 1: Docker Compose (Easiest)

```bash
cd /Users/vakulkumar/.gemini/antigravity/scratch/inventory-microservices

# Start all services
docker compose up --build -d

# Wait 30 seconds for services to initialize

# Verify services are healthy
docker compose ps
```

### Option 2: Kubernetes with Minikube

```bash
# Install Minikube
brew install minikube

# Start Minikube
minikube start --memory=4096 --cpus=2

# Build Docker images
cd services/inventory-service && docker build -t inventory-service:latest . && cd ../..
cd services/order-service && docker build -t order-service:latest . && cd ../..
cd services/notification-service && docker build -t notification-service:latest . && cd ../..
cd services/api-gateway && docker build -t api-gateway:latest . && cd ../..

# Load images into Minikube
minikube image load inventory-service:latest
minikube image load order-service:latest
minikube image load notification-service:latest
minikube image load api-gateway:latest

# Deploy to Kubernetes
kubectl apply -f k8s/

# Wait for pods to be ready
kubectl get pods -w

# Access services
kubectl port-forward svc/api-gateway 8080:8080 &
kubectl port-forward svc/prometheus 9090:9090 &
kubectl port-forward svc/grafana 3000:3000 &
```

## Testing the System

### Automated Tests

```bash
./test-system.sh
```

### Manual Tests

```bash
# Create a product
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gaming Laptop",
    "description": "High-end gaming laptop with RTX 4090",
    "price": 2499.99,
    "stock": 25
  }'

# List all products
curl http://localhost:8080/api/products | jq '.'

# Place an order
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "quantity": 3
  }' | jq '.'

# View all orders
curl http://localhost:8080/api/orders | jq '.'

# Check updated stock
curl http://localhost:8080/api/products/1 | jq '.stock'
```

## Accessing Dashboards

### Prometheus
- URL: http://localhost:9090
- Try these queries:
  - `rate(inventory_http_requests_total[1m])`
  - `histogram_quantile(0.95, sum(rate(order_http_request_duration_seconds_bucket[1m])) by (le))`
  - `inventory_stock_levels`
  - `sum(rate(notification_notifications_sent_total[1m])) by (event_type)`

### Grafana
- URL: http://localhost:3000
- Username: `admin`
- Password: `admin`
- Dashboard: "Microservices Inventory System - Overview"

## Monitoring Events

```bash
# Watch Kafka events in notification service
docker compose logs -f notification-service

# You should see emojis and event messages like:
# 📦 NOTIFICATION: New product added!
# 📧 NOTIFICATION: New order created!
# 🔄 NOTIFICATION: Product updated!
# ⚠️  ALERT: Low stock warning!
```

## Load Testing

```bash
# Install hey (HTTP load generator)
brew install hey

# Generate 1000 requests with 50 concurrent workers
hey -n 1000 -c 50 http://localhost:8080/api/products

# Watch metrics update in real-time in Grafana!
```

## Stopping Services

```bash
# Docker Compose
docker compose down

# Kubernetes
kubectl delete -f k8s/
minikube stop
```

## Troubleshooting

### Services won't start
```bash
# Check logs
docker compose logs <service-name>

# Restart specific service
docker compose restart <service-name>

# Rebuild and restart
docker compose up --build <service-name>
```

### Database connection issues
```bash
# Check database health
docker compose ps

# View database logs
docker compose logs inventory-db
docker compose logs order-db

# Restart databases
docker compose restart inventory-db order-db
```

### Kafka issues
```bash
# Check Kafka logs
docker compose logs kafka

# Verify topics exist
docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
```

### Metrics not appearing
```bash
# Check Prometheus targets
# Visit: http://localhost:9090/targets
# All services should show as "UP"

# Test metrics endpoint directly
curl http://localhost:8081/metrics
curl http://localhost:8082/metrics
curl http://localhost:8083/metrics
curl http://localhost:8080/metrics
```

## Project Location

```
/Users/vakulkumar/.gemini/antigravity/scratch/inventory-microservices
```

Set this as your workspace for easy navigation.

## Key Files

- **README.md** - Comprehensive documentation
- **docker-compose.yml** - Docker Compose configuration
- **test-system.sh** - Automated testing script
- **k8s/** - Kubernetes manifests
- **services/** - All microservice source code
- **monitoring/** - Prometheus and Grafana configs

## Next Steps

1. Install Docker Desktop
2. Run `docker compose up --build -d`
3. Execute `./test-system.sh`
4. Open Grafana at http://localhost:3000
5. Explore metrics and create custom queries!

Enjoy your microservices system! 🚀
