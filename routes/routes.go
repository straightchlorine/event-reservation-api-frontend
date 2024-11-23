package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
	"event-reservation-api/routes/handlers"
)

func SetupRoutes(pool *pgxpool.Pool, jwtSecret string) *mux.Router {
	r := mux.NewRouter()

	// Public routes (no auth required)
	r.HandleFunc("/login", handlers.LoginHandler(pool, jwtSecret)).Methods(http.MethodPost)

	userRouter := r.PathPrefix("/api/users").Subrouter()

	// Authentication required for user routes
	authMiddleware := middlewares.RequireAuth(jwtSecret)
	userRouter.Use(authMiddleware)
	userRouter.HandleFunc("", handlers.GetUserHandler(pool)).Methods(http.MethodGet)
	userRouter.HandleFunc("/{id}", handlers.GetUserByIDHandler(pool)).Methods(http.MethodGet)

	userRouter.HandleFunc("/", handlers.CreateUserHandler(pool)).Methods(http.MethodPut)
	userRouter.HandleFunc("/{id}", handlers.DeleteUserHandler(pool)).Methods(http.MethodDelete)
	userRouter.HandleFunc("/{id}", handlers.UpdateUserHandler(pool)).Methods(http.MethodPut)

	return r
}
