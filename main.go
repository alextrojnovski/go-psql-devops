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
		log.Fatal("❌ DB open error:", err)
	}

	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("⏳ Waiting for DB (%d/5): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("❌ DB ping failed:", err)
	}

	createTable()

	startTime = time.Now()
	log.Println("✅ Server started on :8080")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/ping-db", dbPingHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func createTable() {
	query := `CREATE TABLE IF NOT EXISTS requests_log (
        id SERIAL PRIMARY KEY,
        created_at TIMESTAMP DEFAULT NOW()
    );`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("⚠️ Table creation warning: %v", err)
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
	db.QueryRow("SELECT COUNT(*) FROM requests_log").Scan(&count)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"db":                    "up",
		"total_requests_logged": count,
	})
}
