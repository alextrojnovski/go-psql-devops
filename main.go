package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type StatusResponse struct {
	Status       string `json:"status"`
	Uptime       string `json:"uptime"`
	DBConnected  bool   `json:"db_connected"`
	RequestCount int64  `json:"request_count"`
	Time         string `json:"time"`
}

var (
	db           *sql.DB
	requestCount int64
	startTime    time.Time

	totalRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_app_http_requests_total",
			Help: "Total number of HTTP requests received by the application.",
		},
		[]string{"method", "endpoint"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_app_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	dbErrorCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "go_app_db_errors_total",
			Help: "Total number of database ping errors.",
		},
	)

	dbUpStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_app_db_up",
			Help: "Shows whether the database is up (1) or down (0).",
		},
	)
)

func main() {
	_ = godotenv.Load()

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "appdb"),
	)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB open error:", err)
	}

	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf(" Waiting for DB (%d/5): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Println(" Running WITHOUT database connection (ping failed)")
	} else {
		createTable()
	}

	startTime = time.Now()
	log.Println(" Server started on :8080")
	go func() {
		for {
			isDBUp := db != nil && db.Ping() == nil
			if isDBUp {
				dbUpStatus.Set(1)
			} else {
				dbUpStatus.Set(0)
				dbErrorCount.Inc()
			}
			time.Sleep(10 * time.Second)
		}
	}()

	metricsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			totalRequests.WithLabelValues(r.Method, r.URL.Path).Inc()
			next(w, r)
			duration := time.Since(start).Seconds()
			requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		}
	}

	http.HandleFunc("/health", metricsMiddleware(healthHandler))
	http.HandleFunc("/status", metricsMiddleware(statusHandler))
	http.HandleFunc("/ping-db", metricsMiddleware(dbPingHandler))
        http.HandleFunc("/version", metricsMiddleware(versionHandler))
	http.Handle("/metrics", promhttp.Handler())

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
    version := os.Getenv("APP_VERSION")
    if version == "" {
        version = "dev"
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"version": version})
}

func createTable() {
	query := `CREATE TABLE IF NOT EXISTS requests_log (
		id SERIAL PRIMARY KEY,
		created_at TIMESTAMP DEFAULT NOW()
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf(" Table creation warning: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	requestCount++

	dbOk := false
	if db != nil {
		dbOk = db.Ping() == nil
		if dbOk {
			go func() {
				_, _ = db.Exec("INSERT INTO requests_log (created_at) VALUES (NOW())")
			}()
		}
	}

	resp := StatusResponse{
		Status:       "running",
		Uptime:       time.Since(startTime).String(),
		DBConnected:  dbOk,
		RequestCount: requestCount,
		Time:         time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func dbPingHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"db": "not_initialized"})
		return
	}

	err := db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"db": "down", "error": err.Error()})
		return
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM requests_log").Scan(&count)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"db":                    "up",
		"total_requests_logged": count,
	})
}
