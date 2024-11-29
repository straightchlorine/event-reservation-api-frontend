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

	// Middlewares
	authMiddleware := middlewares.RequireAuth(jwtSecret)
	tokenValidationMiddleware := middlewares.TokenValidation(pool, jwtSecret)

	// Public routes
	setupPublicRoutes(r, pool, jwtSecret)

	// Protected routes
	setupLocationRoutes(r, pool, authMiddleware, tokenValidationMiddleware)
	setupReservationRoutes(r, pool, authMiddleware, tokenValidationMiddleware)
	setupEventRoutes(r, pool, authMiddleware, tokenValidationMiddleware)
	setupUserRoutes(r, pool, authMiddleware, tokenValidationMiddleware)

	return r
}

func setupPublicRoutes(r *mux.Router, pool *pgxpool.Pool, jwtSecret string) {
	r.HandleFunc("/api/login", handlers.LoginHandler(pool, jwtSecret)).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", handlers.LogoutHandler(pool, jwtSecret)).Methods(http.MethodPost)

	r.HandleFunc("/api/events", handlers.GetEventsHandler(pool)).Methods(http.MethodGet)
	r.HandleFunc("/api/events/{id}", handlers.GetEventByIDHandler(pool)).Methods(http.MethodGet)
	r.HandleFunc("/api/locations", handlers.GetLocationsHandler(pool)).Methods(http.MethodGet)
	r.HandleFunc("/api/locations/{id}", handlers.GetLocationByIDHandler(pool)).
		Methods(http.MethodGet)
}

func setupLocationRoutes(
	r *mux.Router,
	pool *pgxpool.Pool,
	authMiddleware, tokenValidationMiddleware mux.MiddlewareFunc,
) {
	locRouter := r.PathPrefix("/api/locations").Subrouter()
	locRouter.Use(authMiddleware, tokenValidationMiddleware)

	locRouter.HandleFunc("", handlers.CreateLocationHandler(pool)).Methods(http.MethodPut)
	locRouter.HandleFunc("/{id}", handlers.UpdateLocationHandler(pool)).Methods(http.MethodPut)
	locRouter.HandleFunc("/{id}", handlers.DeleteLocationHandler(pool)).Methods(http.MethodDelete)
}

func setupReservationRoutes(
	r *mux.Router,
	pool *pgxpool.Pool,
	authMiddleware, tokenValidationMiddleware mux.MiddlewareFunc,
) {
	resRouter := r.PathPrefix("/api/reservations").Subrouter()
	resRouter.Use(authMiddleware, tokenValidationMiddleware)

	resRouter.HandleFunc("", handlers.CreateReservationHandler(pool)).Methods(http.MethodPut)
	resRouter.HandleFunc("", handlers.GetReservationHandler(pool)).Methods(http.MethodGet)
	resRouter.HandleFunc("/{id}", handlers.GetReservationByIDHandler(pool)).Methods(http.MethodGet)
	resRouter.HandleFunc("/{id}/tickets", handlers.GetReservationByIDHandler(pool)).
		Methods(http.MethodGet)
	resRouter.HandleFunc("/{id}", handlers.DeleteReservationHandler(pool)).
		Methods(http.MethodDelete)
	resRouter.HandleFunc("/user", handlers.GetCurrentUserReservationsHandler(pool)).
		Methods(http.MethodGet)

	// Uncomment and implement when ready
	// resRouter.HandleFunc("/{id}", handlers.UpdateReservationHandler(pool)).Methods(http.MethodPut)
}

func setupEventRoutes(
	r *mux.Router,
	pool *pgxpool.Pool,
	authMiddleware, tokenValidationMiddleware mux.MiddlewareFunc,
) {
	eventRouter := r.PathPrefix("/api/events").Subrouter()
	eventRouter.Use(authMiddleware, tokenValidationMiddleware)

	eventRouter.HandleFunc("", handlers.CreateEventHandler(pool)).Methods(http.MethodPut)
	eventRouter.HandleFunc("/{id}", handlers.UpdateEventHandler(pool)).Methods(http.MethodPut)
	eventRouter.HandleFunc("/{id}", handlers.DeleteEventHandler(pool)).Methods(http.MethodDelete)
}

func setupUserRoutes(
	r *mux.Router,
	pool *pgxpool.Pool,
	authMiddleware, tokenValidationMiddleware mux.MiddlewareFunc,
) {
	userRouter := r.PathPrefix("/api/users").Subrouter()
	userRouter.Use(authMiddleware, tokenValidationMiddleware)

	userRouter.HandleFunc("", handlers.GetUserHandler(pool)).Methods(http.MethodGet)
	userRouter.HandleFunc("/{id}", handlers.GetUserByIDHandler(pool)).Methods(http.MethodGet)
	userRouter.HandleFunc("/", handlers.CreateUserHandler(pool)).Methods(http.MethodPut)
	userRouter.HandleFunc("/{id}", handlers.DeleteUserHandler(pool)).Methods(http.MethodDelete)
	userRouter.HandleFunc("/{id}", handlers.UpdateUserHandler(pool)).Methods(http.MethodPut)
}
