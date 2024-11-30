package db

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"event-reservation-api/models"
)

// Fetch the IDs of existing records from given table.
func fetchIds(ctx context.Context, pool *pgxpool.Pool, table string) []int {
	// fetch the ids from the table
	rows, err := pool.Query(ctx, "SELECT id FROM "+table)
	if err != nil {
		return []int{}
	}
	defer rows.Close()

	// build the list of ids
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return []int{}
		}
		ids = append(ids, id)
	}
	return ids
}

// Fetch the IDs of existing records from given table.
func fetchUUIDIds(ctx context.Context, pool *pgxpool.Pool, table string) []uuid.UUID {
	// fetch the ids from the table
	rows, err := pool.Query(ctx, "SELECT id FROM "+table)
	if err != nil {
		return []uuid.UUID{}
	}
	defer rows.Close()

	// build the list of ids
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return []uuid.UUID{}
		}
		ids = append(ids, id)
	}
	return ids
}

// Add an admin user to the database.
func AddAdminUser(fake *gofakeit.Faker, pool *pgxpool.Pool) error {
	// get the root user credentials from env
	root_password := os.Getenv("ROOT_PASSWORD")
	root_username := os.Getenv("ROOT_NAME")

	// if not provided, go with root
	if root_password == "" {
		root_password = "root"
	}
	if root_username == "" {
		root_username = "root"
	}

	// create hash from provided password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(root_password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if fake == nil {
		fake = gofakeit.New(0)
	}

	// fill in the root user struct
	root_user := UserPopulate{
		Name:     "Root",
		Surname:  "Root",
		Username: root_username,
		Email:    "root@root.rt",
		RoleID:   3, // admin role
		IsActive: true,
	}

	// query template for inserting a user
	query := `
		INSERT INTO Users (name, surname, username,
											email, password_hash, role_id,
											is_active, last_login, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`

	// in case nil is passed, populating is off, so run regular insert
	if pool != nil {
		_, err := pool.Query(
			context.Background(),
			query,
			root_user.Name,
			root_user.Surname,
			root_user.Username,
			root_user.Email,
			string(passwordHash),
			root_user.RoleID,
			root_user.IsActive,
		)
		if err != nil {
			return err
		}

	}
	return nil
}

type UserPopulate struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	RoleID   int    `json:"role_id"`
	IsActive bool   `json:"is_active"`
}

// Populate the database with fake user records.
func populateUsers(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// user struct
	users := make([]UserPopulate, 40)

	// role struct
	roleIDs := fetchIds(ctx, pool, "Roles")

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range users {
		// fake password, and its hash
		password := fake.Password(true, true, true, true, false, 12)
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// fill the struct
		users[i] = UserPopulate{
			Name:     fake.FirstName(),
			Surname:  fake.LastName(),
			Username: fake.Username(),
			Email:    fake.Email(),
			RoleID:   roleIDs[i%len(roleIDs)],
			IsActive: fake.Bool(),
		}

		// fill the batch with requests
		batch.Queue(
			`INSERT INTO Users (name, surname, username,
                          email, password_hash, role_id,
													is_active, last_login, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			users[i].Name,
			users[i].Surname,
			users[i].Username,
			users[i].Email,
			string(passwordHash),
			users[i].RoleID,
			users[i].IsActive,
			fake.PastDate(),
			fake.PastDate(),
		)
	}

	// if possible, take advantage of the batch request
	err := AddAdminUser(fake, pool)
	if err != nil {
		return err
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

// Populate the database with fake location records.
func populateLocations(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// locations struct
	locations := make([]models.CreateLocationRequest, 20)

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range locations {
		fake_stadium := fake.NounCollectiveThing()

		locations[i] = models.CreateLocationRequest{
			Stadium:  strings.ToUpper(string(fake_stadium[0])) + fake_stadium[1:],
			Address:  fake.Address().Address,
			Country:  fake.Country(),
			Capacity: fake.Number(5000, 80000),
		}

		batch.Queue(
			`INSERT INTO Locations (stadium, address, country, capacity)
        VALUES ($1, $2, $3, $4)`,
			locations[i].Stadium,
			locations[i].Address,
			locations[i].Country,
			locations[i].Capacity,
		)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

type EventPopulate struct {
	Name             string    `json:"name"`
	Date             time.Time `json:"date"`
	AvailableTickets int       `json:"available_tickets"`
	Price            float64   `json:"price"`
	LocationID       int       `json:"location_id"`
}

// Populate the database with fake event records.
func populateEvents(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// get existing location ids
	locationIDs := fetchIds(ctx, pool, "Locations")

	// event struct
	events := make([]EventPopulate, 20)

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range events {
		events[i] = EventPopulate{
			Name:             fake.Country() + "-" + fake.Country(),
			Date:             fake.FutureDate(),
			Price:            fake.Price(10, 1000),
			LocationID:       locationIDs[i%len(locationIDs)],
			AvailableTickets: fake.Number(1000, 50000),
		}

		// fill the batch with requests
		batch.Queue(
			`INSERT INTO Events (name, date, price, location_id, available_tickets)
        VALUES ($1, $2, $3, $4, $5)`,
			events[i].Name,
			events[i].Date,
			events[i].Price,
			events[i].LocationID,
			events[i].AvailableTickets,
		)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

type ReservationPopulate struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	EventID      int       `json:"event_id"`
	CreatedAt    time.Time `json:"created_at"`
	TotalTickets int       `json:"total_tickets"`
	StatusID     int       `json:"status_id"`
}

// Populate the database with fake reservations.
func populateReservations(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// fetch user ids
	userIDs := fetchUUIDIds(ctx, pool, "Users")

	// fetch event ids
	eventIDs := fetchIds(ctx, pool, "Events")

	// fetch status ids
	statusIDs := fetchIds(ctx, pool, "reservation_statuses")

	// reservation struct
	reservations := make([]ReservationPopulate, 100)

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range reservations {

		// fill the struct
		reservations[i] = ReservationPopulate{
			UserID:       userIDs[i%len(userIDs)],
			EventID:      eventIDs[i%len(eventIDs)],
			CreatedAt:    fake.PastDate(),
			TotalTickets: fake.Number(1, 10),
			StatusID:     statusIDs[i%len(statusIDs)],
		}

		// fill the batch with requests
		batch.Queue(
			`INSERT INTO Reservations (user_id, event_id, created_at, total_tickets, status_id)
        VALUES ($1, $2, $3, $4, $5)`,
			reservations[i].UserID,
			reservations[i].EventID,
			reservations[i].CreatedAt,
			reservations[i].TotalTickets,
			reservations[i].StatusID,
		)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

type TicketPopulate struct {
	ID            int       `json:"id"`
	ReservationID uuid.UUID `json:"reservation_id"`
	Price         float64   `json:"price"`
	TypeID        int       `json:"type_id"`
	StatusID      int       `json:"status_id"`
}

// Populate the database with fake tickets.
func populateTickets(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// ticket struct
	tickets := make([]TicketPopulate, 100)

	// fetch reservation ids
	reservationIDs := fetchUUIDIds(ctx, pool, "Reservations")

	// fetch ticket type ids
	ticketTypeIDs := fetchIds(ctx, pool, "ticket_types")

	// fetch ticket status ids
	ticketStatusIDs := fetchIds(ctx, pool, "ticket_statuses")

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range tickets {

		// fill the struct
		tickets[i] = TicketPopulate{
			ReservationID: reservationIDs[i%len(reservationIDs)],
			Price:         fake.Price(10, 1000),
			TypeID:        ticketTypeIDs[i%len(ticketTypeIDs)],
			StatusID:      ticketStatusIDs[i%len(ticketStatusIDs)],
		}

		// fill the batch with requests
		batch.Queue(
			`INSERT INTO Tickets (reservation_id, price, type_id, status_id)
        VALUES ($1, $2, $3, $4)`,
			tickets[i].ReservationID,
			tickets[i].Price,
			tickets[i].TypeID,
			tickets[i].StatusID,
		)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

// Populate the database with fake data.
func PopulateDatabase(pool *pgxpool.Pool) error {
	// conte  // context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// group the functions in appropriate order
	populationFuncs := []func(context.Context, *pgxpool.Pool) error{
		populateUsers,
		populateLocations,
		populateEvents,
		populateReservations,
		populateTickets,
	}

	// populate the database
	for _, populateFunc := range populationFuncs {
		if err := populateFunc(ctx, pool); err != nil {
			return err
		}
	}

	return nil
}
