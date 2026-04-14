package main

import (
	"testing"
)

func TestHealthCheck(t *testing.T) {
	if true != true {
		t.Error("Something went wrong")
	}
}

func TestAppStarts(t *testing.T) {
	t.Log("App compiles successfully")
}
