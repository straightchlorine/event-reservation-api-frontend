package routes

import (
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
	"event-reservation-api/routes/handlers"
)

func SetupRoutes(pool *pgxpool.Pool, jwtSecret string) *mux.Router {
	r := mux.NewRouter()

	authMiddleware := middlewares.RequireAuth(jwtSecret)

	// Public routes (no auth required)
	r.HandleFunc("/login", handlers.LoginHandler(pool, jwtSecret)).Methods("POST")

	userRouter := r.PathPrefix("/api/users").Subrouter()
	userRouter.Use(authMiddleware)
	userRouter.HandleFunc("/", handlers.GetUserHandler(pool)).Methods("GET")

	return r
}
