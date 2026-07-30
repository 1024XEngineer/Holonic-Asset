package main

import "testing"

func TestInitServerBuildsApplication(t *testing.T) {
	app := InitServer()
	if app == nil {
		t.Fatal("expected server application")
	}
	if app.engine == nil {
		t.Fatal("expected server engine")
	}
}
