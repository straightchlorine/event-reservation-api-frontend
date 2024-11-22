package db

import (
	"context"
	"strings"
	"time"

	"event-reservation-api/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

/*
Fetch the IDs of existing records from given table.

	Arguments:
	  ctx: context
	  pool: database connection pool
	  table: table name

	Returns:
	  []int: list of integer IDs
*/
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

/*
Fetch the IDs of existing records from given table.

	Arguments:
	  ctx: context
	  pool: database connection pool
	  table: table name

	Returns:
	  []uuid.UUID: list of UUIDs
*/
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

// Populate the database with fake user records.
func populateUsers(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// user struct
	users := make([]models.User, 40)

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
		users[i] = models.User{
			Name:         fake.FirstName(),
			Surname:      fake.LastName(),
			Username:     fake.Username(),
			Email:        fake.Email(),
			LastLogin:    fake.PastDate(),
			CreatedAt:    fake.PastDate(),
			PasswordHash: string(passwordHash),
			RoleID:       roleIDs[i%len(roleIDs)],
			IsActive:     fake.Bool(),
		}

		// fill the batch with requests
		batch.Queue(`
			INSERT INTO Users (name, surname, username,
                         email, last_login, created_at,
                         password_hash, role_id, is_active)
			VALUES ($1, $2, $3,
              $4, $5, $6,
              $7, $8, $9)
		`, users[i].Name, users[i].Surname, users[i].Username,
			users[i].Email, users[i].LastLogin, users[i].CreatedAt,
			users[i].PasswordHash, users[i].RoleID, users[i].IsActive)
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
	locations := make([]models.Location, 20)

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range locations {
		fake_stadium := fake.NounCollectiveThing()

		locations[i] = models.Location{
			Stadium:  strings.ToUpper(string(fake_stadium[0])) + fake_stadium[1:],
			Address:  fake.Address().Address,
			Country:  fake.Country(),
			Capacity: fake.Number(5000, 80000),
		}

		batch.Queue(`
      INSERT INTO Locations (stadium, address, country, capacity)
      VALUES ($1, $2, $3, $4)
    `, locations[i].Stadium, locations[i].Address, locations[i].Country, locations[i].Capacity)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

// Populate the database with fake event records.
func populateEvents(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// get existing location ids
	locationIDs := fetchIds(ctx, pool, "Locations")

	// event struct
	events := make([]models.Event, 20)

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range events {
		events[i] = models.Event{
			Name:             fake.Country() + "-" + fake.Country(),
			Date:             fake.FutureDate(),
			LocationID:       locationIDs[i%len(locationIDs)],
			AvailableTickets: fake.Number(1000, 50000),
		}

		// fill the batch with requests
		batch.Queue(`
				INSERT INTO Events (name, date, location_id, available_tickets)
				VALUES ($1, $2, $3, $4)
			`, events[i].Name, events[i].Date, events[i].LocationID, events[i].AvailableTickets)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

// Populate the database with fake reservations.
func populateReservations(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// fetch user ids
	userIDs := fetchIds(ctx, pool, "Users")

	// fetch event ids
	eventIDs := fetchIds(ctx, pool, "Events")

	// fetch status ids
	statusIDs := fetchIds(ctx, pool, "ReservationStatuses")

	// reservation struct
	reservations := make([]models.Reservation, 100)

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range reservations {

		// fill the struct
		reservations[i] = models.Reservation{
			PrimaryUserID: userIDs[i%len(userIDs)],
			EventID:       eventIDs[i%len(eventIDs)],
			CreatedAt:     fake.PastDate(),
			TotalTickets:  fake.Number(1, 10),
			StatusID:      statusIDs[i%len(statusIDs)],
		}

		reservation_id := uuid.New()

		// fill the batch with requests
		batch.Queue(`
        INSERT INTO Reservations (id, primary_user_id, event_id, created_at, total_tickets, status_id)
        VALUES ($1, $2, $3, $4, $5, $6)
      `, reservation_id, reservations[i].PrimaryUserID, reservations[i].EventID,
			reservations[i].CreatedAt, reservations[i].TotalTickets, reservations[i].StatusID)

	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

// Populate the database with fake tickets.
func populateTickets(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// ticket struct
	tickets := make([]models.Ticket, 100)

	//fetch event ids
	eventIDs := fetchIds(ctx, pool, "Events")

	// fetch reservation ids
	reservationIDs := fetchUUIDIds(ctx, pool, "Reservations")

	// fetch ticket type ids
	ticketTypeIDs := fetchIds(ctx, pool, "TicketTypes")

	// fetch ticket status ids
	ticketStatusIDs := fetchIds(ctx, pool, "TicketStatuses")

	// batch insert
	batch := &pgx.Batch{}

	// fill the struct with fake data
	for i := range tickets {

		// fill the struct
		tickets[i] = models.Ticket{
			EventID:       eventIDs[i%len(eventIDs)],
			ReservationID: reservationIDs[i%len(reservationIDs)],
			Price:         fake.Price(10, 1000),
			TypeID:        ticketTypeIDs[i%len(ticketTypeIDs)],
			StatusID:      ticketStatusIDs[i%len(ticketStatusIDs)],
			SeatNumber:    fake.Word(),
		}

		// fill the batch with requests
		batch.Queue(`
              INSERT INTO Tickets (event_id, reservation_id, price, type_id, status_id, seat_number)
              VALUES ($1, $2, $3, $4, $5, $6)
            `, tickets[i].EventID, tickets[i].ReservationID, tickets[i].Price,
			tickets[i].TypeID, tickets[i].StatusID, tickets[i].SeatNumber)
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
