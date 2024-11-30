# Ticket Reservation API

This project is a university exercise designed to simulate a ticket reservation system for sports events. **It does not include payment processing or other real-world functionalities.**

---

## Features

- **RESTful API** for managing:
  - Users
  - Events
  - Locations
  - Reservations
- **Authentication** for routes with sensitive data.
- **NocoDB** for visual database exploration.
- **Swagger Documentation** for testing and exploring endpoints.

## Prerequisites

- Install [Docker Compose](https://docs.docker.com/compose/).

---

## Setup

1. Clone the repository:
   ```bash
   git clone git@github.com:straightchlorine/event-reservation-api.git && cd event-reservation-api
   ```

2. Configure environment variables:

    Use the provided `.env` file and modify it as needed.

3. Build and run the application:
   ```bash
   docker-compose build && docker-compose up
   ```

## Services

Utilising provided `.env`:

- **API:** [http://localhost:8080](http://localhost:8080/api/login)
- **NocoDB:** [http://localhost:8081](http://localhost:8081)
- **Swagger Documentation:** [http://localhost:80](http://localhost:80/docs)

## API Endpoints

### Events
- `GET /events` - Retrieve all events.
- `PUT /events` - Create a new event (admin).
- `DELETE /events/{id}` - Delete an event (admin).
- `GET /events/{id}` - Retrieve an event by ID.
- `PUT /events/{id}` - Update an event (admin).

### Locations
- `GET /locations` - Retrieve all locations.
- `PUT /locations` - Create a new location (admin).
- `DELETE /locations/{id}` - Delete a location (admin).
- `GET /locations/{id}` - Retrieve a location by ID.
- `PUT /locations/{id}` - Update a location (admin).

### Authentication
- `POST /login` - Log in to the API.
- `POST /logout` - Log out from the API.

### Reservations
- `GET /reservations` - List all reservations (admin).
- `DELETE /reservations/{id}` - Delete a reservation by ID (admin).
- `GET /reservations/{id}` - Retrieve a reservation by ID (admin/resource owner).
- `POST /reservations/{id}/cancel` - Cancel a reservation (admin/resource owner).
- `GET /reservations/{id}/tickets` - List tickets for a reservation (admin/resource owner).
- `PUT /reservations` - Create a reservation (at least registered).
- `GET /reservations/user` - List reservations for the current user.
- `GET /reservations/user/{id}` - List reservations for a user by ID (admin/resource owner).
- `GET /reservations/user/{id}/tickets` - List tickets for a user by ID (admin/resource owner).
- `GET /reservations/user/tickets` - List tickets for the current user.

### Users
- `GET /users` - List all users (admin).
- `PUT /users` - Create a new user.
- `DELETE /users/{id}` - Delete a user by ID (admin/resource owner).
- `GET /users/{id}` - Retrieve a user by ID (admin).
- `PUT /users/{id}` - Update a user by ID.

---

## Environment Variables

| **Variable Name**       | **Description**                                   | **Default Value**      |
|-------------------------|---------------------------------------------------|------------------------|
| `DB_HOST`               | PostgreSQL database host                          | `database`             |
| `DB_PORT`               | PostgreSQL database port                          | `5432`                 |
| `DB_USER`               | Database username                                 | `postgres`             |
| `DB_PASSWORD`           | Database password                                 | `password`             |
| `DB_NAME`               | Name of the PostgreSQL database                  | `event_api`            |
| `NC_DB_USER`            | NocoDB database username                          | `nocodb`               |
| `NC_DB_PASSWORD`        | NocoDB database password                          | `password`             |
| `NC_DB_NAME`            | NocoDB database name                              | `nocodb_metadata`      |
| `NC_DB_PORT`            | NocoDB database port                              | `8081`                 |
| `NC_AUTH_JWT`           | JWT secret for NocoDB authentication              | `nocodb-jwt-secret`    |
| `API_PORT`              | API server port                                   | `8080`                 |
| `API_JWT_SECRET`        | JWT secret for API authentication                 | `api-secret`           |
| `API_ROOT_NAME`         | Admin username for API setup                      | `root`                 |
| `API_ROOT_PASSWORD`     | Admin password for API setup                      | `root`                 |
| `API_TOKEN_VALID_HOURS` | Token validity duration (in hours)                | `24`                   |
| `SWAGGER_PORT`          | Port for serving Swagger documentation            | `80`                   |

---

## Notes

- **Authentication:** Many routes require authentication with role-based permissions (e.g., admin, owner).
- **Dynamic IDs:** Routes using `{id}` operate on a specific resource identified by its ID.
