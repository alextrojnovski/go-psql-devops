package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestStatusHandler(t *testing.T) {
	req, _ := http.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statusHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := response["status"]; !ok {
		t.Error("Response missing 'status' field")
	}
	if _, ok := response["time"]; !ok {
		t.Error("Response missing 'time' field")
	}
}

func TestDbPingHandlerWithoutDB(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = nil

	req, _ := http.NewRequest("GET", "/ping-db", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(dbPingHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %v", status)
	}
}

func TestDbPingHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

}

func BenchmarkHealthHandler(b *testing.B) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rr, req)
	}
}
