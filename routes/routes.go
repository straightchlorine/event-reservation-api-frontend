package routes

import (
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Group and initialize all routes
func SetupRoutes(pool *pgxpool.Pool) *mux.Router {
	r := mux.NewRouter()

	// register routes
	// RegisterEventRoutes(r, pool)
	// RegisterReservationRoutes(r, pool)

	return r
}
