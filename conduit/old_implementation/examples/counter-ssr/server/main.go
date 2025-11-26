package main

import (
	"log"
	"os"

	"github.com/user/gox/runtime/server"
	counter "github.com/user/gox/dist" // Import the compiled Counter component
)

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server
	srv := server.New(port)

	// Register routes
	srv.Route("/", func() server.Component {
		// Create Counter with initial value 0
		return counter.NewCounter(0)
	})

	srv.Route("/counter", func() server.Component {
		// Create Counter with initial value 10
		return counter.NewCounter(10)
	})

	// Serve static files (CSS, JS, images)
	srv.Static("/static/", "./static")

	// Start server
	log.Printf("Starting GoX server on port %s", port)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}