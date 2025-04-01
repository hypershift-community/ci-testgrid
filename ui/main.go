package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/hypershift-community/ci-testgrid/ui/testgrid"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	// Create a new testgrid handler
	handler, err := testgrid.NewHandler(templateFS)
	if err != nil {
		log.Fatalf("Error creating testgrid handler: %v", err)
	}

	// Set up routes
	http.Handle("/", handler)

	// Start the server
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
