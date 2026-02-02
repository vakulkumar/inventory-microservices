package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

// Payment represents a payment record
type Payment struct {
	ID        int       `json:"id"`
	OrderID   int       `json:"order_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Prometheus metrics
var (
	paymentsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_processed_total",
			Help: "Total number of payments processed",
		},
		[]string{"status"},
	)
	paymentProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "payment_processing_duration_seconds",
			Help:    "Payment processing time in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
)

var db *sql.DB
var kafkaWriter *kafka.Writer

func main() {
	// Database connection
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432") // Default to standard postgres port if not set
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "payment_db")

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

	// Kafka Producer Setup
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    "payment-events",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// Kafka Consumer Setup
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    "order-events",
		GroupID:  "payment-service",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Start consuming messages
	go consumeMessages(ctx, reader)

	// HTTP Server
	router := mux.NewRouter()
	router.Use(metricsMiddleware)

	router.HandleFunc("/payments", getPayments).Methods("GET")
	router.HandleFunc("/payments/{id}", getPayment).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	port := getEnv("PORT", "8084")
	log.Printf("Payment Service starting on port %s", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Stopping HTTP server...")

	// Create a deadline to wait for.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

	reader.Close()
	log.Println("Payment Service stopped")
}

func initDB() {
	schema := `
	CREATE TABLE IF NOT EXISTS payments (
		id SERIAL PRIMARY KEY,
		order_id INTEGER NOT NULL,
		amount DECIMAL(10, 2) NOT NULL,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatal("Failed to create schema:", err)
	}
	log.Println("Database schema initialized")
}

func consumeMessages(ctx context.Context, reader *kafka.Reader) {
	log.Println("Started consuming order-events...")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			var event map[string]interface{}
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			eventType, ok := event["event_type"].(string)
			if !ok {
				continue
			}

			if eventType == "order_created" {
				processPayment(event)
			}
		}
	}
}

func processPayment(event map[string]interface{}) {
	start := time.Now()

	// Extract details safely
	orderIDFloat, _ := event["order_id"].(float64)
	amount, _ := event["total_price"].(float64)
	orderID := int(orderIDFloat)

	log.Printf("Processing payment for Order ID: %d, Amount: %.2f", orderID, amount)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Create payment record
	var paymentID int
	var createdAt time.Time
	status := "completed" // Mock success

	err := db.QueryRow(
		"INSERT INTO payments (order_id, amount, status) VALUES ($1, $2, $3) RETURNING id, created_at",
		orderID, amount, status,
	).Scan(&paymentID, &createdAt)

	if err != nil {
		log.Printf("Failed to save payment: %v", err)
		paymentsProcessed.WithLabelValues("failed").Inc()
		return
	}

	// Publish Payment Processed Event
	paymentEvent := map[string]interface{}{
		"event_type": "payment_processed",
		"payment_id": paymentID,
		"order_id":   orderID,
		"amount":     amount,
		"status":     status,
		"timestamp":  time.Now().Unix(),
	}

	publishEvent(paymentEvent)

	paymentsProcessed.WithLabelValues("success").Inc()
	paymentProcessingDuration.Observe(time.Since(start).Seconds())
	log.Printf("Payment processed successfully. Payment ID: %d", paymentID)
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

func getPayments(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, order_id, amount, status, created_at FROM payments ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	payments := []Payment{}
	for rows.Next() {
		var p Payment
		err := rows.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Status, &p.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payments = append(payments, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

func getPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var p Payment
	err := db.QueryRow("SELECT id, order_id, amount, status, created_at FROM payments WHERE id = $1", id).
		Scan(&p.ID, &p.OrderID, &p.Amount, &p.Status, &p.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
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

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
