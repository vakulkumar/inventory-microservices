# Architecture Documentation

## System Overview

This document details the architectural decisions, patterns, and design principles used in the Microservices Inventory System.

## Architecture Principles

### 1. Separation of Concerns
Each microservice has a single, well-defined responsibility:
- **Inventory Service** - Product catalog management only
- **Order Service** - Order processing only
- **Notification Service** - Event notifications only
- **API Gateway** - Request routing only

### 2. Database Per Service Pattern

**Why**: 
- Data isolation prevents tight coupling
- Each service can choose optimal schema design
- Independent scaling of data layer
- No cross-service database queries

**Implementation**:
```
Inventory Service → PostgreSQL (inventory_db)
Order Service     → PostgreSQL (order_db)
```

**Trade-offs**:
- ✅ Benefits: Loose coupling, independent deployments
- ❌ Challenges: Distributed transactions, data consistency

### 3. Event-Driven Architecture

**Why**:
- Asynchronous communication reduces coupling
- Services don't need to know about each other
- Easy to add new consumers without modifying producers
- Built-in audit trail and event sourcing

**Implementation**:
```
Inventory Service --[publishes]--> Kafka --[consumes]--> Notification Service
Order Service     --[publishes]--> Kafka --[consumes]--> Notification Service
```

**Kafka Topics**:
- `inventory-events` - Product lifecycle events
- `order-events` - Order lifecycle events

**Event Types**:
- `product_created`, `product_updated`, `product_deleted`
- `order_created`
- `low_stock_alert`

### 4. API Gateway Pattern

**Why**:
- Single entry point for clients
- Centralized cross-cutting concerns (logging, rate limiting)
- Simplifies client-side logic
- Enables gradual migration to microservices

**Implementation**:
```
Client → API Gateway → Route to appropriate microservice
```

**Routing Logic**:
- `/api/products/*` → Inventory Service
- `/api/orders/*` → Order Service

## Service Communication Patterns

### Synchronous Communication (HTTP REST)

Used when immediate response is required:

```
Order Service → Inventory Service (GET product info, UPDATE stock)
API Gateway → All Services (routing requests)
```

**When to use**:
- Need immediate response
- Request-response pattern
- Simple queries

### Asynchronous Communication (Kafka)

Used for fire-and-forget events:

```
Inventory Service → Kafka → Notification Service
Order Service → Kafka → Notification Service
```

**When to use**:
- Don't need immediate response
- Broadcasting to multiple consumers
- Event sourcing and audit trails

## Data Flow Examples

### Example 1: Create Product

```
1. Client → API Gateway: POST /api/products
2. API Gateway → Inventory Service: POST /products
3. Inventory Service:
   - Validates data
   - Inserts into PostgreSQL
   - Publishes "product_created" to Kafka
   - Updates Prometheus metrics
4. Kafka → Notification Service: Consumes event
5. Notification Service: Logs notification
6. API Gateway → Client: Returns created product
```

### Example 2: Place Order

```
1. Client → API Gateway: POST /api/orders
2. API Gateway → Order Service: POST /orders
3. Order Service → Inventory Service: GET /products/{id}
4. Inventory Service → Order Service: Returns product details
5. Order Service:
   - Validates stock availability
   - Creates order in PostgreSQL
   - Calls Inventory Service: PUT /products/{id} (reduce stock)
   - Publishes "order_created" to Kafka
   - Updates Prometheus metrics
6. Kafka → Notification Service: Consumes event
7. Notification Service: Logs order notification
8. API Gateway → Client: Returns order confirmation
```

## Observability Strategy

### Metrics Collection

**Philosophy**: Measure everything that matters to business and operations

**Implementation**:
- Every service exports metrics on `/metrics` endpoint
- Prometheus scrapes every 15 seconds
- Metrics stored with labels for filtering

**Metric Types**:

1. **Counter** - Monotonically increasing values
   - `inventory_http_requests_total`
   - `order_orders_total`
   - `notification_notifications_sent_total`

2. **Histogram** - Distribution of values
   - `inventory_http_request_duration_seconds`
   - `order_processing_duration_seconds`

3. **Gauge** - Current value that can go up or down
   - `inventory_stock_levels`

### Custom Business Metrics

**Why**: Technical metrics alone don't tell the business story

**Examples**:
- `inventory_stock_levels` - Track inventory in real-time
- `order_orders_total{status="confirmed"}` - Business KPI
- `notification_notifications_sent_total{event_type="low_stock_alert"}` - Critical alerts

### Grafana Dashboards

**Design Principles**:
- **Top-down approach**: Overview → Detailed metrics
- **Color coding**: Green (good), Yellow (warning), Red (critical)
- **Percentiles over averages**: P95, P99 more useful than mean
- **Business + Technical**: Mix KPIs with system metrics

**Dashboard Sections**:
1. Request rate and latency (technical)
2. Error rates (technical)
3. Order processing (business)
4. Stock levels (business)
5. Database performance (technical)
6. Notification distribution (business)

## Scalability Considerations

### Horizontal Scaling

**Services**:
- Inventory Service: 2 replicas (can scale to handle more reads)
- Order Service: 2 replicas (distribute order processing)
- API Gateway: 2 replicas (high availability)
- Notification Service: 1 replica (can scale if needed)

**Databases**:
- PostgreSQL: Currently single instance
- Future: Read replicas for Inventory Service queries

**Kafka**:
- Currently 1 broker (local dev)
- Production: 3+ brokers for fault tolerance

### Vertical Scaling

**Resource Limits** (Future):
```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Kafka Consumer Groups

**Pattern**: Multiple consumers in same group = load balancing
```
notification-service (group: notification-service)
  Consumer 1 ─┐
  Consumer 2 ─┼─→ Kafka Topic (partitioned)
  Consumer 3 ─┘
```

## Resilience Patterns

### Health Checks

**Liveness Probe**: Is the service alive?
- Checks: HTTP GET /health returns 200
- Failure: Restart pod

**Readiness Probe**: Is the service ready for traffic?
- Checks: HTTP GET /health + database ping
- Failure: Remove from load balancer

### Database Connection Retry

```go
for i := 0; i < 30; i++ {
    err = db.Ping()
    if err == nil {
        break
    }
    time.Sleep(2 * time.Second)
}
```

**Why**: Databases may not be ready immediately on startup

### Graceful Shutdown

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
// Cleanup: close Kafka connections, database, etc.
```

**Why**: Prevents data loss and incomplete transactions

## Security Considerations

### Current State (Development)

- ⚠️ No authentication/authorization
- ⚠️ Plain text credentials in environment variables
- ⚠️ No TLS/SSL encryption

### Production Recommendations

1. **API Authentication**: Add JWT tokens
2. **Service-to-Service Auth**: mTLS
3. **Secrets Management**: Kubernetes Secrets, Vault
4. **Database Encryption**: TLS connections, encrypted storage
5. **Network Policies**: Limit service-to-service communication

## Performance Optimizations

### Database Connection Pooling

```go
db, _ := sql.Open("postgres", connStr)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### HTTP Client Timeouts

```go
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

**Why**: Prevents hanging requests and cascading failures

### Kafka Batching

```go
MinBytes: 10e3,  // Wait for 10KB before consuming
MaxBytes: 10e6,  // Max 10MB per fetch
```

**Why**: Reduces network overhead for high-throughput scenarios

## Technology Choices Rationale

### Why Go?

- ✅ Fast compilation and runtime performance
- ✅ Built-in concurrency (goroutines)
- ✅ Small binary size (Docker images)
- ✅ Great HTTP and networking libraries
- ✅ Strong type safety

### Why PostgreSQL?

- ✅ ACID compliance (important for orders)
- ✅ Rich query capabilities
- ✅ JSON support (flexible schema evolution)
- ✅ Proven reliability

### Why Kafka?

- ✅ High throughput, low latency
- ✅ Persistent message storage
- ✅ Horizontal scalability
- ✅ Strong ordering guarantees
- ✅ Industry standard

### Why Prometheus + Grafana?

- ✅ De facto standard for Kubernetes monitoring
- ✅ Powerful query language (PromQL)
- ✅ Pull-based model (services expose, Prometheus scrapes)
- ✅ Beautiful, flexible dashboards

## Future Enhancements

### 1. Distributed Tracing

**Tool**: Jaeger or Zipkin
**Why**: Track requests across multiple services
**Example**: See complete journey of an order from API Gateway → Order Service → Inventory Service

### 2. Circuit Breaker

**Tool**: resilience4go or custom implementation
**Why**: Prevent cascading failures
**Example**: If Inventory Service is down, fail fast instead of timing out

### 3. Service Mesh

**Tool**: Istio or Linkerd
**Why**: 
- Automatic mTLS
- Traffic management
- Advanced load balancing
- Observability out-of-the-box

### 4. API Rate Limiting

**Implementation**: Token bucket algorithm
**Why**: Prevent abuse and ensure fair usage

### 5. Caching Layer

**Tool**: Redis
**Why**: Reduce database load for frequently accessed products

### 6. Saga Pattern

**Why**: Distributed transactions across services
**Example**: Order creation + Payment + Inventory update

## Deployment Strategies

### Blue-Green Deployment

```
Blue (old version)  ─┐
                     ├──→ Load Balancer
Green (new version) ─┘
```

### Canary Deployment

```
v1.0 (90% traffic) ─┐
                    ├──→ Load Balancer
v2.0 (10% traffic) ─┘
```

### Rolling Update (Current)

```
Pod 1: v1 → v2
Pod 2: v1 → v2
Pod 3: v1 → v2
```

Kubernetes handles this automatically with `kubectl apply`.

## Monitoring and Alerting

### Key Metrics to Alert On

1. **Error Rate > 1%**
   - `rate(inventory_http_requests_total{status=~"5.."}[5m]) > 0.01`

2. **Latency P95 > 1s**
   - `histogram_quantile(0.95, rate(order_http_request_duration_seconds_bucket[5m])) > 1`

3. **Low Stock**
   - `inventory_stock_levels < 10`

4. **Service Down**
   - `up{job="inventory-service"} == 0`

### Alert Routing

```
Prometheus → Alertmanager → Slack/PagerDuty/Email
```

## Conclusion

This architecture demonstrates:
- **Scalable**: Horizontal scaling with Kubernetes
- **Resilient**: Health checks, retries, graceful shutdown
- **Observable**: Comprehensive metrics and dashboards
- **Maintainable**: Clear separation of concerns
- **Evolvable**: Event-driven design allows easy extension

The focus on **observability** distinguishes this implementation, providing deep insights into both system performance and business metrics.
