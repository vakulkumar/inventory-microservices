package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

// Prometheus metrics
var (
	notificationsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_notifications_sent_total",
			Help: "Total number of notifications sent",
		},
		[]string{"event_type"},
	)
	messageProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "notification_message_processing_duration_seconds",
			Help:    "Message processing time in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func main() {
	// Kafka consumer setup
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")

	// Read from multiple topics
	topics := []string{"inventory-events", "order-events", "payment-events"}

	readers := make([]*kafka.Reader, len(topics))
	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{kafkaBroker},
			Topic:    topic,
			GroupID:  "notification-service",
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		})
	}

	// Start Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", healthCheck)
		port := getEnv("PORT", "8083")
		log.Printf("Metrics server starting on port %s", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

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

	// Start consuming from all topics
	for i, reader := range readers {
		go consumeMessages(ctx, reader, topics[i])
	}

	log.Println("Notification Service started, waiting for messages...")
	<-ctx.Done()

	// Close all readers
	for _, reader := range readers {
		reader.Close()
	}
	log.Println("Notification Service stopped")
}

func consumeMessages(ctx context.Context, reader *kafka.Reader, topic string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()

			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("Error reading message from %s: %v", topic, err)
				continue
			}

			// Parse message
			var event map[string]interface{}
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			// Process notification
			eventType := event["event_type"].(string)
			processNotification(event, eventType)

			notificationsSent.WithLabelValues(eventType).Inc()
			messageProcessingDuration.Observe(time.Since(start).Seconds())
		}
	}
}

func processNotification(event map[string]interface{}, eventType string) {
	switch eventType {
	case "order_created":
		log.Printf("📧 NOTIFICATION: New order created! Order ID: %.0f, Product ID: %.0f, Quantity: %.0f",
			event["order_id"], event["product_id"], event["quantity"])

	case "product_created":
		log.Printf("📦 NOTIFICATION: New product added! Product ID: %.0f, Name: %s",
			event["product_id"], event["name"])

	case "product_updated":
		log.Printf("🔄 NOTIFICATION: Product updated! Product ID: %s, Name: %s, Stock: %.0f",
			event["product_id"], event["name"], event["stock"])

	case "low_stock_alert":
		log.Printf("⚠️  ALERT: Low stock warning! Product ID: %s, Name: %s, Remaining stock: %.0f",
			event["product_id"], event["name"], event["stock"])

	case "product_deleted":
		log.Printf("🗑️  NOTIFICATION: Product deleted! Product ID: %s",
			event["product_id"])

	case "payment_processed":
		log.Printf("💸 NOTIFICATION: Payment processed! Payment ID: %.0f, Order ID: %.0f, Amount: %.2f, Status: %s",
			event["payment_id"], event["order_id"], event["amount"], event["status"])

	default:
		log.Printf("📨 NOTIFICATION: Unknown event type: %s", eventType)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
