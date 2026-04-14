//go:build integration
// +build integration

package main

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func TestDatabaseIntegration(t *testing.T) {
	connStr := "host=" + os.Getenv("DB_HOST") +
		" port=" + os.Getenv("DB_PORT") +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" sslmode=disable"
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	
	err = db.Ping()
	if err != nil {
		t.Fatal(err)
	}
	
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY)")
	if err != nil {
		t.Fatal(err)
	}
	
	_, err = db.Exec("INSERT INTO test_table DEFAULT VALUES")
	if err != nil {
		t.Fatal(err)
	}
	
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	
	if count < 1 {
		t.Error("Expected at least 1 row")
	}
}
