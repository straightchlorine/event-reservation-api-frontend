package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"event-reservation-api/db"
	"event-reservation-api/routes"
)

func main() {
	// get the connection pool
	pool, err := db.Connect()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// set up the routes
	r := routes.SetupRoutes(pool)

	// start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
