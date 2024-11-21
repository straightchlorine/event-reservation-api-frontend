// Main application runner.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"event-reservation-api/db"
	"event-reservation-api/routes"
)

// Main function
// Handles populating the database and exposing the API routes.
func main() {

	populateFlag := flag.Bool("populate", false, "Populate the database with initial data.")

	// Parse the command line flags.
	flag.Parse()

	// Get the connection pool.
	pool, err := db.Connect()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// If population is set, run the population script.
	if *populateFlag {
		fmt.Println("Populating the database...")
		err = db.PopulateDatabase(pool)
		if err != nil {
			log.Fatalf("Failed to populate the database: %v\n", err)
		}
		fmt.Println("Database populated successfully.")
	}

	// Set up the API routes.
	r := routes.SetupRoutes(pool)

	// Start the server (defaults to port 8080).
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Log the server start.
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
