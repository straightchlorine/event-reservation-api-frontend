package db

import (
	"context"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// TODO: refactor already existing code, and use the models

// Populate the database with fake user records.
func populateUsers(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// get existing role ids
	rows, err := pool.Query(ctx, "SELECT id FROM Roles")
	if err != nil {
		return err
	}
	defer rows.Close()

	// build the list of role ids
	var roleIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		roleIDs = append(roleIDs, id)
	}

	// user struct
	users := make([]struct {
		Username     string
		Email        string
		Name         string
		Surname      string
		RoleID       int
		PasswordHash []byte
	}, 20)

	// fill the struct with fake data
	batch := &pgx.Batch{}
	for i := range users {
		// fake password, and its hash
		password := fake.Password(true, true, true, true, false, 12)
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// fill the struct
		users[i] = struct {
			Username     string
			Email        string
			Name         string
			Surname      string
			RoleID       int
			PasswordHash []byte
		}{
			Username:     fake.Username(),
			Email:        fake.Email(),
			Name:         fake.FirstName(),
			Surname:      fake.LastName(),
			RoleID:       roleIDs[i%len(roleIDs)],
			PasswordHash: passwordHash,
		}

		// insert into database via pgx.Batch for bulk insert
		batch.Queue(`
			INSERT INTO Users (username, email, password_hash, name, surname, role_id, is_active, email_verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, users[i].Username, users[i].Email, users[i].PasswordHash,
			users[i].Name, users[i].Surname, users[i].RoleID, true, true)
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

	// location struct
	locations := make([]struct {
		Stadium  string
		Address  string
		Country  string
		Capacity int
	}, 5)

	// fill the struct with fake data
	for i := range locations {
		fake_stadium := fake.NounCollectiveThing()
		locations[i] = struct {
			Stadium  string
			Address  string
			Country  string
			Capacity int
		}{
			Stadium:  strings.ToUpper(string(fake_stadium[0])) + fake_stadium[1:],
			Address:  fake.Address().Address,
			Country:  fake.Country(),
			Capacity: fake.Number(5000, 80000),
		}
	}

	// insert into database via pgx.Batch for bulk insert
	batch := &pgx.Batch{}
	for _, loc := range locations {
		batch.Queue(`
			INSERT INTO Locations (stadium, address, country, capacity)
			VALUES ($1, $2, $3, $4)
		`, loc.Stadium, loc.Address, loc.Country, loc.Capacity)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	return nil
}

// Populate the database with fake event records.
func populateEvents(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	// get existing location ids
	rows, err := pool.Query(ctx, "SELECT id FROM Locations")
	if err != nil {
		return err
	}
	defer rows.Close()

	// build the list of location ids
	var locationIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		locationIDs = append(locationIDs, id)
	}

	// event struct
	events := make([]struct {
		Name             string
		Date             time.Time
		LocationID       int
		AvailableTickets int
	}, 10)

	// fill the struct with fake data
	for i := range events {
		events[i] = struct {
			Name             string
			Date             time.Time
			LocationID       int
			AvailableTickets int
		}{
			Name:             fake.Country() + "-" + fake.Country(),
			Date:             fake.FutureDate(),
			LocationID:       locationIDs[i%len(locationIDs)],
			AvailableTickets: fake.Number(1000, 50000),
		}
	}

	// insert into database via pgx.Batch for bulk insert
	batch := &pgx.Batch{}
	for _, event := range events {
		batch.Queue(`
			INSERT INTO Events (name, date, location_id, available_tickets)
			VALUES ($1, $2, $3, $4)
		`, event.Name, event.Date, event.LocationID, event.AvailableTickets)
	}

	// send the batch
	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	// return nil if no errors
	return nil
}

// TODO: append additional methods as they will be written.
// Populate the database with fake group orders.
func populateGroupOrders(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	var userIDs, eventIDs, statusIDs []int

	// fetch user ids
	userRows, err := pool.Query(ctx, "SELECT id FROM Users")
	if err != nil {
		return err
	}
	defer userRows.Close()

	// build the list of user ids
	for userRows.Next() {
		var id int
		if err := userRows.Scan(&id); err != nil {
			return err
		}
		userIDs = append(userIDs, id)
	}

	// fetch event ids
	eventRows, err := pool.Query(ctx, "SELECT id FROM Events")
	if err != nil {
		return err
	}
	defer eventRows.Close()

	// build the list of event ids
	for eventRows.Next() {
		var id int
		if err := eventRows.Scan(&id); err != nil {
			return err
		}
		eventIDs = append(eventIDs, id)
	}

	// fetch the reservation statuses
	statusRows, err := pool.Query(ctx, "SELECT id FROM ReservationStatuses")
	if err != nil {
		return err
	}
	defer statusRows.Close()

	// bild the list of status ids
	for statusRows.Next() {
		var id int
		if err := statusRows.Scan(&id); err != nil {
			return err
		}
		statusIDs = append(statusIDs, id)
	}

	groupOrders := make([]struct {
		PrimaryUserID int
		EventID       int
		TotalTickets  int
		StatusID      int
	}, 15)

	batch := &pgx.Batch{}
	for i := range groupOrders {
		groupOrders[i] = struct {
			PrimaryUserID int
			EventID       int
			TotalTickets  int
			StatusID      int
		}{
			PrimaryUserID: userIDs[i%len(userIDs)],
			EventID:       eventIDs[i%len(eventIDs)],
			TotalTickets:  fake.Number(1, 10),
			StatusID:      statusIDs[i%len(statusIDs)],
		}

		groupOrderID := uuid.New()
		batch.Queue(`
			INSERT INTO GroupOrders (id, primary_user_id, event_id, total_tickets, status_id)
			VALUES ($1, $2, $3, $4, $5)
		`, groupOrderID, groupOrders[i].PrimaryUserID,
			groupOrders[i].EventID, groupOrders[i].TotalTickets, groupOrders[i].StatusID)

		// Add group order participants
		participantCount := groupOrders[i].TotalTickets
		for j := 0; j < participantCount; j++ {
			batch.Queue(`
				INSERT INTO GroupOrderParticipants (group_order_id, name, email)
				VALUES ($1, $2, $3)
			`, groupOrderID, fake.Name(), fake.Email())
		}
	}

	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	return nil
}

// Fetch the ID fields from Event table.
func fetchEventIds(pool *pgxpool.Pool) []int {

	// list placeholder
	var eventIDs []int

	// fetch event ids
	eventRows, err := pool.Query(ctx, "SELECT id FROM Events")
	if err != nil {
		return err
	}
	defer eventRows.Close()

	// build the list of event ids
	for eventRows.Next() {
		var id int
		if err := eventRows.Scan(&id); err != nil {
			return err
		}
		eventIDs = append(eventIDs, id)
	}

	return eventIDs
}

// TODO: finish implementation
func populateTickets(ctx context.Context, pool *pgxpool.Pool) error {
	fake := gofakeit.New(0)

	eventIds := fetchEventIds(pool)

	return nil
}

// Populate the database with fake data.
func PopulateDatabase(pool *pgxpool.Pool) error {

	// Create a context with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Population order matters due to foreign key constraints
	populationFuncs := []func(context.Context, *pgxpool.Pool) error{
		populateLocations,
		populateEvents,
		populateUsers,
		populateGroupOrders,
	}

	for _, populateFunc := range populationFuncs {
		if err := populateFunc(ctx, pool); err != nil {
			return err
		}
	}

	return nil
}
