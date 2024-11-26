-- Event Ticketing System Database Schema
-- Drop existing tables
DROP TABLE IF EXISTS Payment CASCADE;

DROP TABLE IF EXISTS Tickets CASCADE;

DROP TABLE IF EXISTS Reservations CASCADE;

DROP TABLE IF EXISTS RolePermissions CASCADE;

DROP TABLE IF EXISTS Permissions CASCADE;

DROP TABLE IF EXISTS UserAuthLogs CASCADE;

DROP TABLE IF EXISTS Users CASCADE;

DROP TABLE IF EXISTS Roles CASCADE;

DROP TABLE IF EXISTS ReservationStatuses CASCADE;

DROP TABLE IF EXISTS PaymentStatuses CASCADE;

DROP TABLE IF EXISTS TicketStatuses CASCADE;

DROP TABLE IF EXISTS TicketTypes CASCADE;

DROP TABLE IF EXISTS Events CASCADE;

DROP TABLE IF EXISTS Locations CASCADE;

-- Load pgcrypto extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Roles of the user within the system
CREATE TABLE Roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE, -- 'UNREGISTERED', 'REGISTERED', 'ADMIN'
  description TEXT
);

-- Users Table
CREATE TABLE Users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  surname VARCHAR(100) NOT NULL,
  username VARCHAR(100) UNIQUE NOT NULL,
  email VARCHAR(150) UNIQUE NOT NULL,
  last_login TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  password_hash VARCHAR(255) NOT NULL,
  role_id INT NOT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  CONSTRAINT fk_user_role FOREIGN KEY (role_id) REFERENCES Roles (id) ON DELETE RESTRICT
);

-- User Authentication Logs, to track login attempts
CREATE TABLE UserAuthLogs (
  id SERIAL PRIMARY KEY,
  user_id INT,
  login_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ip_address INET,
  user_agent TEXT,
  login_status BOOLEAN NOT NULL,
  CONSTRAINT fk_user_auth_log FOREIGN KEY (user_id) REFERENCES Users (id) ON DELETE CASCADE
);

-- Permissions for each role
CREATE TABLE Permissions (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) UNIQUE NOT NULL,
  description TEXT
);

-- Mapping Permissions onto Roles in a many-to-many relationship
CREATE TABLE RolePermissions (
  role_id INT,
  permission_id INT,
  PRIMARY KEY (role_id, permission_id),
  CONSTRAINT fk_role_permission_role FOREIGN KEY (role_id) REFERENCES Roles (id) ON DELETE CASCADE,
  CONSTRAINT fk_role_permission_permission FOREIGN KEY (permission_id) REFERENCES Permissions (id) ON DELETE CASCADE
);

-- Locations Table
CREATE TABLE Locations (
  id SERIAL PRIMARY KEY,
  stadium VARCHAR(150) NOT NULL,
  address VARCHAR(250) NOT NULL,
  country VARCHAR(100) NOT NULL,
  capacity INT NOT NULL CHECK (capacity > 0)
);

-- Event Table
CREATE TABLE Events (
  id SERIAL PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  date TIMESTAMP NOT NULL CHECK (date > CURRENT_TIMESTAMP),
  price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
  location_id INT NOT NULL,
  available_tickets INT NOT NULL CHECK (available_tickets >= 0),
  CONSTRAINT fk_event_location FOREIGN KEY (location_id) REFERENCES Locations (id) ON DELETE CASCADE
);

-- Statuses for Reservations
CREATE TABLE ReservationStatuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- Reservations Table
CREATE TABLE Reservations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
  primary_user_id INT NOT NULL,
  event_id INT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  total_tickets INT NOT NULL CHECK (total_tickets > 0),
  status_id INT NOT NULL,
  CONSTRAINT fk_reservation_user FOREIGN KEY (primary_user_id) REFERENCES Users (id) ON DELETE CASCADE,
  CONSTRAINT fk_reservation_event FOREIGN KEY (event_id) REFERENCES Events (id) ON DELETE CASCADE,
  CONSTRAINT fk_reservation_status FOREIGN KEY (status_id) REFERENCES ReservationStatuses (id) ON DELETE CASCADE
);

-- Ticket Types
CREATE TABLE TicketTypes (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  discount DECIMAL(5, 2) NOT NULL CHECK (discount BETWEEN 0 AND 1),
  description VARCHAR(250) NOT NULL
);

-- Ticket Statuses
CREATE TABLE TicketStatuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- Tickets Table
CREATE TABLE Tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
  event_id INT NOT NULL,
  reservation_id UUID,
  price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
  type_id INT NOT NULL,
  status_id INT NOT NULL,
  CONSTRAINT fk_ticket_event FOREIGN KEY (event_id) REFERENCES Events (id) ON DELETE CASCADE,
  CONSTRAINT fk_ticket_reservation_id FOREIGN KEY (reservation_id) REFERENCES Reservations (id) ON DELETE CASCADE,
  CONSTRAINT fk_ticket_type FOREIGN KEY (type_id) REFERENCES TicketTypes (id) ON DELETE CASCADE,
  CONSTRAINT fk_ticket_status FOREIGN KEY (status_id) REFERENCES TicketStatuses (id) ON DELETE CASCADE
);

-- Payment Statuses
CREATE TABLE PaymentStatuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- Payments Table
CREATE TABLE Payment (
  id SERIAL PRIMARY KEY,
  order_id UUID NOT NULL,
  status_id INT NOT NULL,
  total_amount DECIMAL(10, 2) NOT NULL,
  payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_payment_reservation FOREIGN KEY (order_id) REFERENCES Reservations (id) ON DELETE CASCADE,
  CONSTRAINT fk_payment_status FOREIGN KEY (status_id) REFERENCES PaymentStatuses (id) ON DELETE CASCADE
);

COMMENT ON TABLE Users IS 'Stores user account information with role-based access';

COMMENT ON TABLE Roles IS 'Defines user roles with different access levels';

COMMENT ON TABLE Permissions IS 'Defines system-wide permissions for different actions';

COMMENT ON TABLE Reservations IS 'Groups tickets into a single order for an event';

COMMENT ON TABLE Tickets IS 'Represents individual tickets within group orders';

-- Initial values for Roles
INSERT INTO
  Roles (name, description)
VALUES
  (
    'UNREGISTERED',
    'Limited access, cannot create reservations'
  ),
  (
    'REGISTERED',
    'Standard user with booking capabilities'
  ),
  ('ADMIN', 'Full system access and management');

-- Initial values for Permissions
INSERT INTO
  Permissions (name, description)
VALUES
  ('VIEW_EVENTS', 'Can view available events'),
  (
    'CREATE_RESERVATION',
    'Can create ticket reservations'
  ),
  ('MANAGE_OWN_PROFILE', 'Can update own profile'),
  (
    'MANAGE_USERS',
    'Can create, update, delete users'
  ),
  (
    'MANAGE_EVENTS',
    'Can create, update, delete events'
  ),
  ('VIEW_REPORTS', 'Can access system reports');

-- Mapping the initial permissions to roles
INSERT INTO
  RolePermissions (role_id, permission_id)
SELECT
  r.id,
  p.id
FROM
  Roles r,
  Permissions p
WHERE
  (
    r.name = 'UNREGISTERED'
    AND p.name IN ('VIEW_EVENTS')
  )
  OR (
    r.name = 'REGISTERED'
    AND p.name IN (
      'VIEW_EVENTS',
      'CREATE_RESERVATION',
      'MANAGE_OWN_PROFILE'
    )
  )
  OR (
    r.name = 'ADMIN'
    AND p.name IN (
      'VIEW_EVENTS',
      'CREATE_RESERVATION',
      'MANAGE_OWN_PROFILE',
      'MANAGE_USERS',
      'MANAGE_EVENTS',
      'VIEW_REPORTS'
    )
  );

-- Initial Reservation Statuses
INSERT INTO
  ReservationStatuses (name)
VALUES
  ('PENDING'),
  ('CONFIRMED'),
  ('CANCELLED');

-- Initial Ticket Statuses
INSERT INTO
  TicketStatuses (name)
VALUES
  ('AVAILABLE'),
  ('RESERVED'),
  ('SOLD'),
  ('CANCELLED');

-- Initial Payment Statuses
INSERT INTO
  PaymentStatuses (name)
VALUES
  ('PENDING'),
  ('COMPLETED'),
  ('FAILED'),
  ('REFUNDED');

-- Initial Ticket Types
INSERT INTO
  TicketTypes (name, discount, description)
VALUES
  ('STANDARD', 0.00, 'Regular price ticket'),
  ('STUDENT', 0.20, '20% discount for students'),
  ('SENIOR', 0.15, '15% discount for seniors');
