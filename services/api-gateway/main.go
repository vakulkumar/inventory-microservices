package main

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"os"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	errorRate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_errors_total",
			Help: "Total number of errors",
		},
		[]string{"route", "error_type"},
	)
)

var inventoryServiceURL string
var orderServiceURL string

var httpClient *http.Client

func init() {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.IdleConnTimeout = 90 * time.Second

	httpClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: t,
	}
}

func main() {
	inventoryServiceURL = getEnv("INVENTORY_SERVICE_URL", "http://localhost:8081")
	orderServiceURL = getEnv("ORDER_SERVICE_URL", "http://localhost:8082")

	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.Use(metricsMiddleware)

	// Route to inventory service
	router.PathPrefix("/api/products").HandlerFunc(proxyToInventory)

	// Route to order service
	router.PathPrefix("/api/orders").HandlerFunc(proxyToOrders)

	// Health check
	router.HandleFunc("/health", healthCheck).Methods("GET")

	// Metrics
	router.Handle("/metrics", promhttp.Handler())

	port := getEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	log.Printf("Routing /api/products -> %s", inventoryServiceURL)
	log.Printf("Routing /api/orders -> %s", orderServiceURL)

	log.Fatal(http.ListenAndServe(":"+port, router))
}

func proxyToInventory(w http.ResponseWriter, r *http.Request) {
	proxyRequest(w, r, inventoryServiceURL, "/api/products", "/products")
}

func proxyToOrders(w http.ResponseWriter, r *http.Request) {
	proxyRequest(w, r, orderServiceURL, "/api/orders", "/orders")
}

func proxyRequest(w http.ResponseWriter, r *http.Request, targetURL, stripPrefix, newPrefix string) {
	// Build target URL
	path := r.URL.Path
	if stripPrefix != "" {
		path = newPrefix + path[len(stripPrefix):]
	}

	targetURL = targetURL + path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		errorRate.WithLabelValues(r.URL.Path, "request_creation").Inc()
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Execute request
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		errorRate.WithLabelValues(r.URL.Path, "request_execution").Inc()
		log.Printf("Error proxying request to %s: %v", targetURL, err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed in %v", time.Since(start))
	})
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

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy"}`))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
