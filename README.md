# Microservices Inventory System with Observability

A production-ready microservices architecture demonstrating distributed systems patterns with comprehensive observability using **Prometheus** and **Grafana**.

![Architecture](https://img.shields.io/badge/Architecture-Microservices-blue) ![Language](https://img.shields.io/badge/Language-Go-00ADD8) ![Database](https://img.shields.io/badge/Database-PostgreSQL-316192) ![Messaging](https://img.shields.io/badge/Messaging-Kafka-231F20) ![Orchestration](https://img.shields.io/badge/Orchestration-Kubernetes-326CE5) ![Monitoring](https://img.shields.io/badge/Monitoring-Prometheus%20%2B%20Grafana-E6522C)

## Architecture Overview

This system consists of **4 microservices** + **1 frontend** implementing a complete e-commerce platform:

```
┌─────────────┐
│   Browser   │  ← ShopHub Frontend (React)
└──────┬──────┘
       │ :3000
       ▼
┌─────────────────┐
│  API Gateway    │ :8080
│  (Reverse Proxy)│
└────┬────────┬───┘
     │        │
     ▼        ▼
┌────────┐  ┌──────────┐
│Inventory│  │  Order   │
│Service  │◄─┤ Service  │
│:8081    │  │ :8082    │
└────┬────┘  └────┬─────┘
     │            │
     ▼            ▼
   ┌──────────────────┐
   │   Kafka Broker   │
   └────────┬─────────┘
            │
            ▼
   ┌──────────────────┐
   │  Notification    │
   │    Service       │
   │     :8083        │
   └──────────────────┘

         │
         ▼
   ┌──────────────┐      ┌──────────┐
   │  Prometheus  │◄─────┤ Grafana  │
   │    :9090     │      │  :3000   │
   └──────────────┘      └──────────┘
```

### Services

1. **ShopHub Frontend** (Port 3000) - E-commerce web application (React)
2. **API Gateway** (Port 8080) - Single entry point, request routing
3. **Inventory Service** (Port 8081) - Product catalog and stock management
4. **Order Service** (Port 8082) - Order processing and fulfillment
5. **Notification Service** (Port 8083) - Event-driven notifications

### Infrastructure

- **React + Vite** - Modern frontend with premium UI
- **PostgreSQL** - Database per service pattern (Inventory DB, Order DB)
- **Kafka** - Event-driven messaging between services
- **Prometheus** - Metrics collection and storage
- **Grafana** - Metrics visualization and dashboards

## Key Features

### Observability (The "Edge")

✅ **Custom Prometheus Metrics**:
- HTTP request rate and latency per endpoint
- Database query performance
- Order processing throughput
- Stock level monitoring
- Kafka message processing metrics
- Error rates and status code distribution

✅ **Pre-built Grafana Dashboard**:
- Real-time request rate visualization
- 95th percentile latency tracking
- Error rate monitoring
- Stock level graphs
- Event distribution analysis

### Microservices Patterns

- **Database per Service** - Data isolation
- **API Gateway** - Single entry point
- **Event-Driven Architecture** - Kafka pub/sub
- **Health Checks** - Service health monitoring
- **Service Discovery** - Docker/Kubernetes networking

## Prerequisites

- **Docker** & **Docker Compose** (for local development)
- **Go 1.21+** (for local development)
- **Kubernetes** & **Minikube** (for K8s deployment)
- **kubectl** (Kubernetes CLI)

## Quick Start with Docker Compose

### 1. Build and Start All Services

```bash
cd inventory-microservices
docker-compose up --build
```

This will start:
- All 4 microservices
- 2 PostgreSQL databases
- Kafka + Zookeeper
- Prometheus
- Grafana

### 2. Verify Services

```bash
# Check all containers are running
docker-compose ps

# API Gateway health check
curl http://localhost:8080/health

# Inventory Service health check
curl http://localhost:8081/health

# Order Service health check
curl http://localhost:8082/health

# Notification Service health check
curl http://localhost:8083/health
```

### 3. Test the APIs

```bash
# Create a product
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 999.99,
    "stock": 50
  }'

# Get all products
curl http://localhost:8080/api/products

# Create an order
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "quantity": 5
  }'

# Get all orders
curl http://localhost:8080/api/orders
```

### 4. Access the Frontend

**ShopHub E-commerce UI**: http://localhost:3000

```bash
# In a new terminal, start the frontend
cd frontend
npm install  # First time only
npm run dev
```

Features:
- 🛍️ Beautiful product catalog
- 🛒 Interactive shopping cart
- ✨ Amazon/Flipkart-inspired design
- 📱 Mobile responsive
- 🎨 Smooth animations

### 5. Access Observability Tools

**Prometheus**: http://localhost:9090
- Explore raw metrics
- Run PromQL queries
- Example: `rate(inventory_http_requests_total[1m])`

**Grafana**: http://localhost:3000
- **Username**: `admin`
- **Password**: `admin`
- View the pre-configured "Microservices Inventory System - Overview" dashboard

### 5. Monitor Kafka Events

Watch notification service logs to see Kafka events being processed:

```bash
docker-compose logs -f notification-service
```

You should see notifications like:
- 📦 New product added
- 📧 Order created
- 🔄 Product updated
- ⚠️ Low stock alerts

## API Documentation

### Inventory Service API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/products` | List all products |
| GET | `/products/{id}` | Get product by ID |
| POST | `/products` | Create new product |
| PUT | `/products/{id}` | Update product |
| DELETE | `/products/{id}` | Delete product |

**Example Product Object**:
```json
{
  "id": 1,
  "name": "Laptop",
  "description": "High-performance laptop",
  "price": 999.99,
  "stock": 50,
  "created_at": "2026-01-30T06:00:00Z"
}
```

### Order Service API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/orders` | List all orders |
| GET | `/orders/{id}` | Get order by ID |
| POST | `/orders` | Create new order |

**Example Order Request**:
```json
{
  "product_id": 1,
  "quantity": 5
}
```

**Example Order Response**:
```json
{
  "id": 1,
  "product_id": 1,
  "quantity": 5,
  "total_price": 4999.95,
  "status": "confirmed",
  "created_at": "2026-01-30T06:05:00Z"
}
```

## Observability Metrics

### Custom Metrics by Service

**Inventory Service**:
- `inventory_http_requests_total` - HTTP request count
- `inventory_http_request_duration_seconds` - Request latency
- `inventory_db_query_duration_seconds` - Database query time
- `inventory_stock_levels` - Current stock levels per product

**Order Service**:
- `order_http_requests_total` - HTTP request count
- `order_http_request_duration_seconds` - Request latency
- `order_orders_total` - Order count by status
- `order_processing_duration_seconds` - Order processing time

**Notification Service**:
- `notification_notifications_sent_total` - Notifications sent by type
- `notification_message_processing_duration_seconds` - Message processing time

**API Gateway**:
- `gateway_http_requests_total` - HTTP request count by route
- `gateway_http_request_duration_seconds` - Request latency
- `gateway_errors_total` - Error count by type

### Kafka Event Topics

- `inventory-events` - Product lifecycle events
- `order-events` - Order lifecycle events

## Kubernetes Deployment

### 1. Start Minikube

```bash
minikube start --memory=4096 --cpus=2
```

### 2. Build and Load Docker Images

```bash
# Build all service images
cd services/inventory-service && docker build -t inventory-service:latest . && cd ../..
cd services/order-service && docker build -t order-service:latest . && cd ../..
cd services/notification-service && docker build -t notification-service:latest . && cd ../..
cd services/api-gateway && docker build -t api-gateway:latest . && cd ../..

# Load images into Minikube
minikube image load inventory-service:latest
minikube image load order-service:latest
minikube image load notification-service:latest
minikube image load api-gateway:latest
```

### 3. Deploy to Kubernetes

```bash
kubectl apply -f k8s/
```

### 4. Access Services

```bash
# Get service URLs
minikube service api-gateway --url
minikube service prometheus --url
minikube service grafana --url

# Or use port forwarding
kubectl port-forward svc/api-gateway 8080:8080
kubectl port-forward svc/prometheus 9090:9090
kubectl port-forward svc/grafana 3000:3000
```

## Load Testing

Generate load to see metrics in action:

```bash
# Install hey (HTTP load generator)
brew install hey

# Generate 1000 requests with 10 concurrent workers
hey -n 1000 -c 10 http://localhost:8080/api/products

# Create multiple orders
for i in {1..50}; do
  curl -X POST http://localhost:8080/api/orders \
    -H "Content-Type: application/json" \
    -d "{\"product_id\": 1, \"quantity\": 1}"
done
```

Watch the metrics update in real-time in Grafana!

## Development

### Running Services Locally

```bash
# Start dependencies
docker-compose up -d inventory-db order-db kafka zookeeper prometheus grafana

# Run inventory service
cd services/inventory-service
go run main.go

# Run order service (in new terminal)
cd services/order-service
go run main.go

# Run notification service (in new terminal)
cd services/notification-service
go run main.go

# Run API gateway (in new terminal)
cd services/api-gateway
go run main.go
```

## Architecture Decisions

### Why Database Per Service?
- **Data isolation** and independence
- Each service can choose optimal database schema
- Prevents tight coupling between services

### Why Kafka?
- **Asynchronous communication** between services
- **Event sourcing** for audit trails
- **Scalability** - multiple consumers can process events
- **Reliability** - message persistence and replay

### Why Prometheus + Grafana?
- **Industry standard** for metrics collection
- **Powerful query language** (PromQL)
- **Beautiful dashboards** in Grafana
- **Alerting capabilities** (can be extended)

## Troubleshooting

### Services won't start
```bash
# Check logs
docker-compose logs <service-name>

# Restart a specific service
docker-compose restart <service-name>
```

### Database connection issues
```bash
# Verify database is healthy
docker-compose ps

# Check database logs
docker-compose logs inventory-db
docker-compose logs order-db
```

### Kafka connection issues
```bash
# Check Kafka is running
docker-compose logs kafka

# Verify topics
docker-compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
```

### Metrics not appearing in Prometheus
```bash
# Check Prometheus targets
# Open http://localhost:9090/targets
# All services should show as "UP"

# Verify metrics endpoint
curl http://localhost:8081/metrics
```

## Future Enhancements

- [ ] Add circuit breaker pattern with resilience4j/Hystrix
- [ ] Implement distributed tracing with Jaeger
- [ ] Add API authentication (JWT)
- [ ] Implement rate limiting
- [ ] Add Kubernetes Horizontal Pod Autoscaler
- [ ] Set up Prometheus alerting rules
- [ ] Add integration tests
- [ ] Implement saga pattern for distributed transactions

## License

MIT

## Author

Built with ❤️ to demonstrate microservices architecture and observability best practices.
