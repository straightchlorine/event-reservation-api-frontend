# Ticket Reservation API

This is a university project and is meant to facilitate ticket reservation API for sports events.
Project does not facilitate payments or any other real-world features, it is just an exercise.

## Features

- **RESTful API** for managing users, events, locations, and reservations.
- **Authentication Middleware** for secure access to protected routes.
- **PostgreSQL** as the primary database.
- **NocoDB** for exploring data within the database.
- **Swagger Documentation** to explore and test API endpoints.
- **Dockerized** easy to deploy.
- **Hoppscotch** for testing API routes.

### Prerequisites

- [docker-compose](https://docs.docker.com/compose/)

## Setup

1. Clone the repository.
```
git clone git@github.com:straightchlorine/event-reservation-api.git && cd event-reservation-api
```

2. Set environment variables.

You can use the `.env` in the repository, and adjust it to your needs.

3. Build and run the application.

```
docker-compose build && docker-compose up
```

## Environment Variables

Default values represent those in the `.env` file in the repository, you can adjust it to your needs.

| Variable Name          | Description                                                     | Default Value in .env  |
|------------------------|-----------------------------------------------------------------|------------------------|
| **`DB_HOST`**          | Host address of the PostgreSQL database.                        | `database`             |
| **`DB_PORT`**          | Port number for the PostgreSQL database.                        | `5432`                 |
| **`DB_USER`**          | Username for accessing the PostgreSQL database.                 | `postgres`             |
| **`DB_PASSWORD`**      | Password for accessing the PostgreSQL database.                 | `password`             |
| **`DB_NAME`**          | Name of the PostgreSQL database.                                | `event_api`            |
| **`NC_DB_USER`**       | Username for accessing the NocoDB database.                     | `nocodb`               |
| **`NC_DB_PASSWORD`**   | Password for accessing the NocoDB database.                     | `password`             |
| **`NC_DB_NAME`**       | Name of the NocoDB database.                                    | `nocodb_metadata`      |
| **`NC_DB_PORT`**       | Port number for the NocoDB database.                            | `8081`                 |
| **`NC_AUTH_JWT`**      | JWT secret for securing NocoDB authentication.                  | `nocodb-jwt-secret`    |
| **`API_PORT`**         | Port on which the API server will run.                          | `8080`                 |
| **`API_JWT_SECRET`**   | JWT secret for securing API authentication.                     | `api-secret`           |
| **`API_ROOT_NAME`**    | Root username for the API admin setup.                          | `root`                 |
| **`API_ROOT_PASSWORD`**| Root password for the API admin setup.                          | `root`                 |
| **`API_TOKEN_VALID_HOURS`** | Duration (in hours) for which API tokens remain valid.     | `24`                   |
| **`SWAGGER_PORT`**     | Port on which the Swagger will be served.                       | `80`                   |
