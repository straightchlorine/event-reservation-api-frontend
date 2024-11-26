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

	// Authentication middleware
	authMiddleware := middlewares.RequireAuth(jwtSecret)

	// Public routes (no auth required)
	r.HandleFunc("/login", handlers.LoginHandler(pool, jwtSecret)).Methods(http.MethodPost)

	r.HandleFunc("/api/events", handlers.GetEventsHandler(pool)).Methods(http.MethodGet)
	r.HandleFunc("/api/events/{id}", handlers.GetEventByIDHandler(pool)).Methods(http.MethodGet)

	reservationRouter := r.PathPrefix("/api/reservations").Subrouter()
	reservationRouter.Use(authMiddleware)
	reservationRouter.HandleFunc("", handlers.CreateReservationHandler(pool)).
		Methods(http.MethodPut)
	reservationRouter.HandleFunc("", handlers.GetReservationHandler(pool)).Methods(http.MethodGet)
	reservationRouter.HandleFunc("/{id}", handlers.GetReservationByIDHandler(pool)).
		Methods(http.MethodGet)
	reservationRouter.HandleFunc("/{id}", handlers.UpdateReservationHandler(pool)).
		Methods(http.MethodPut)
	reservationRouter.HandleFunc("/{id}", handlers.DeleteReservationHandler(pool)).
		Methods(http.MethodDelete)

	eventRouter := r.PathPrefix("/api/events").Subrouter()
	eventRouter.Use(authMiddleware)
	eventRouter.HandleFunc("", handlers.CreateEventHandler(pool)).Methods(http.MethodPut)
	eventRouter.HandleFunc("/{id}", handlers.UpdateEventHandler(pool)).Methods(http.MethodPut)
	eventRouter.HandleFunc("/{id}", handlers.DeleteEventHandler(pool)).Methods(http.MethodDelete)

	userRouter := r.PathPrefix("/api/users").Subrouter()

	userRouter.Use(authMiddleware)
	userRouter.HandleFunc("", handlers.GetUserHandler(pool)).Methods(http.MethodGet)
	userRouter.HandleFunc("/{id}", handlers.GetUserByIDHandler(pool)).Methods(http.MethodGet)

	userRouter.HandleFunc("/", handlers.CreateUserHandler(pool)).Methods(http.MethodPut)
	userRouter.HandleFunc("/{id}", handlers.DeleteUserHandler(pool)).Methods(http.MethodDelete)
	userRouter.HandleFunc("/{id}", handlers.UpdateUserHandler(pool)).Methods(http.MethodPut)

	return r
}
