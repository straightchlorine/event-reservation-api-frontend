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

No real setup required, just:

```
git clone git@github.com:straightchlorine/event-reservation-api.git && \
cd event-reservation-api && \
docker-compose build && docker-compose up
```

is enough to start the application.
