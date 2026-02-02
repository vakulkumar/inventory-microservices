package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

// Order represents a customer order
type Order struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ProductID  int       `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Product represents product info from inventory service
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// Prometheus metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	ordersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_orders_total",
			Help: "Total number of orders",
		},
		[]string{"status"},
	)
	orderProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "order_processing_duration_seconds",
			Help:    "Order processing time in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

var db *sql.DB
var kafkaWriter *kafka.Writer

func main() {
	// Database connection
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "order_db")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Println("Waiting for database connection...")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Database did not become ready:", err)
	}

	// Initialize database schema
	initDB()

	// Kafka producer
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    "order-events",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// HTTP router
	router := mux.NewRouter()
	router.Use(metricsMiddleware)

	router.HandleFunc("/orders", createOrder).Methods("POST")
	router.HandleFunc("/orders", getOrders).Methods("GET")
	router.HandleFunc("/orders/{id}", getOrder).Methods("GET")
	router.HandleFunc("/orders/user/{userId}", getOrdersByUser).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	port := getEnv("PORT", "8082")
	log.Printf("Order Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func initDB() {
	schema := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL DEFAULT 0,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		total_price DECIMAL(10, 2) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatal("Failed to create schema:", err)
	}

	// Migration for existing table
	_, err = db.Exec("ALTER TABLE orders ADD COLUMN IF NOT EXISTS user_id INTEGER NOT NULL DEFAULT 0;")
	if err != nil {
		log.Println("Warning: Failed to add user_id column (might already exist or other error):", err)
	}

	log.Println("Database schema initialized")
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(wrapped.statusCode)).Inc()
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	var orderReq struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
		UserID    int `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch product info from inventory service
	inventoryURL := getEnv("INVENTORY_SERVICE_URL", "http://localhost:8081")
	product, err := getProductInfo(inventoryURL, orderReq.ProductID)
	if err != nil {
		http.Error(w, "Failed to fetch product info: "+err.Error(), http.StatusInternalServerError)
		ordersTotal.WithLabelValues("failed").Inc()
		return
	}

	// Check stock availability
	if product.Stock < orderReq.Quantity {
		http.Error(w, "Insufficient stock", http.StatusBadRequest)
		ordersTotal.WithLabelValues("failed").Inc()
		return
	}

	// Calculate total price
	totalPrice := product.Price * float64(orderReq.Quantity)

	// Create order
	var order Order
	err = db.QueryRow(
		"INSERT INTO orders (product_id, quantity, total_price, status, user_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at",
		orderReq.ProductID, orderReq.Quantity, totalPrice, "confirmed", orderReq.UserID,
	).Scan(&order.ID, &order.CreatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		ordersTotal.WithLabelValues("failed").Inc()
		return
	}

	order.ProductID = orderReq.ProductID
	order.Quantity = orderReq.Quantity
	order.TotalPrice = totalPrice
	order.Status = "confirmed"
	order.UserID = orderReq.UserID

	// Update inventory (reduce stock)
	newStock := product.Stock - orderReq.Quantity
	err = updateProductStock(inventoryURL, orderReq.ProductID, product, newStock)
	if err != nil {
		log.Printf("Failed to update inventory: %v", err)
	}

	// Publish event to Kafka
	event := map[string]interface{}{
		"event_type":  "order_created",
		"order_id":    order.ID,
		"product_id":  order.ProductID,
		"quantity":    order.Quantity,
		"total_price": order.TotalPrice,
		"timestamp":   time.Now().Unix(),
	}
	publishEvent(event)

	ordersTotal.WithLabelValues("confirmed").Inc()
	orderProcessingDuration.Observe(time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, user_id, product_id, quantity, total_price, status, created_at FROM orders ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		err := rows.Scan(&o.ID, &o.UserID, &o.ProductID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var o Order
	err := db.QueryRow("SELECT id, user_id, product_id, quantity, total_price, status, created_at FROM orders WHERE id = $1", id).
		Scan(&o.ID, &o.UserID, &o.ProductID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func getOrdersByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["userId"]

	rows, err := db.Query("SELECT id, user_id, product_id, quantity, total_price, status, created_at FROM orders WHERE user_id = $1 ORDER BY id DESC", userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		err := rows.Scan(&o.ID, &o.UserID, &o.ProductID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	err := db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getProductInfo(baseURL string, productID int) (*Product, error) {
	url := fmt.Sprintf("%s/products/%d", baseURL, productID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product not found")
	}

	var product Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, err
	}

	return &product, nil
}

func updateProductStock(baseURL string, productID int, product *Product, newStock int) error {
	url := fmt.Sprintf("%s/products/%d", baseURL, productID)
	
	updateData := map[string]interface{}{
		"name":        product.Name,
		"description": "",
		"price":       product.Price,
		"stock":       newStock,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update stock: %s", string(bodyBytes))
	}

	return nil
}

func publishEvent(event map[string]interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Value: data,
	})
	if err != nil {
		log.Printf("Failed to publish event to Kafka: %v", err)
	} else {
		log.Printf("Published event: %s", string(data))
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
