-- Event Ticketing System Database Schema
-- Drop existing tables
DROP TABLE IF EXISTS payment CASCADE;

DROP TABLE IF EXISTS tickets CASCADE;

DROP TABLE IF EXISTS reservations CASCADE;

DROP TABLE IF EXISTS role_permissions CASCADE;

DROP TABLE IF EXISTS permissions CASCADE;

DROP TABLE IF EXISTS user_auth_logs CASCADE;

DROP TABLE IF EXISTS users CASCADE;

DROP TABLE IF EXISTS roles CASCADE;

DROP TABLE IF EXISTS reservation_statuses CASCADE;

DROP TABLE IF EXISTS payment_statuses CASCADE;

DROP TABLE IF EXISTS ticket_statuses CASCADE;

DROP TABLE IF EXISTS ticket_types CASCADE;

DROP TABLE IF EXISTS events CASCADE;

DROP TABLE IF EXISTS locations CASCADE;

-- Load pgcrypto extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Blacklisted tokens for logout
CREATE TABLE token_blacklist (
  id SERIAL PRIMARY KEY,
  token TEXT NOT NULL,
  expires_at TIMESTAMP NOT NULL
);

-- Roles of the user within the system
CREATE TABLE roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE, -- 'UNREGISTERED', 'REGISTERED', 'ADMIN'
  description TEXT
);

-- Users Table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
  name VARCHAR(100) NOT NULL,
  surname VARCHAR(100) NOT NULL,
  username VARCHAR(100) UNIQUE NOT NULL,
  email VARCHAR(150) UNIQUE NOT NULL,
  last_login TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  password_hash VARCHAR(255) NOT NULL,
  role_id INT NOT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  CONSTRAINT fk_user_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE RESTRICT
);

-- User Authentication Logs, to track login attempts
CREATE TABLE user_auth_logs (
  id SERIAL PRIMARY KEY,
  user_id UUID NOT NULL,
  login_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ip_address INET,
  user_agent TEXT,
  login_status BOOLEAN NOT NULL,
  CONSTRAINT fk_user_auth_log FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- permissions for each role
CREATE TABLE permissions (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) UNIQUE NOT NULL,
  description TEXT
);

-- Mapping Permissions onto Roles in a many-to-many relationship
CREATE TABLE role_permissions (
  role_id INT,
  permission_id INT,
  PRIMARY KEY (role_id, permission_id),
  CONSTRAINT fk_role_permission_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
  CONSTRAINT fk_role_permission_permission FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

-- Locations Table
CREATE TABLE locations (
  id SERIAL PRIMARY KEY,
  stadium VARCHAR(150) NOT NULL,
  address VARCHAR(250) NOT NULL,
  country VARCHAR(100) NOT NULL,
  capacity INT NOT NULL CHECK (capacity > 0)
);

-- Event Table
CREATE TABLE events (
  id SERIAL PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  date TIMESTAMP NOT NULL CHECK (date > CURRENT_TIMESTAMP),
  price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
  location_id INT NOT NULL,
  available_tickets INT NOT NULL CHECK (available_tickets >= 0),
  CONSTRAINT fk_event_location FOREIGN KEY (location_id) REFERENCES Locations (id) ON DELETE CASCADE
);

-- Statuses for reservations
CREATE TABLE reservation_statuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- reservations Table
CREATE TABLE reservations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
  user_id UUID NOT NULL,
  event_id INT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  total_tickets INT NOT NULL CHECK (total_tickets > 0),
  status_id INT NOT NULL,
  CONSTRAINT fk_reservation_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_reservation_event FOREIGN KEY (event_id) REFERENCES Events (id) ON DELETE CASCADE,
  CONSTRAINT fk_reservation_status FOREIGN KEY (status_id) REFERENCES reservation_statuses (id) ON DELETE CASCADE
);

-- Ticket Types
CREATE TABLE ticket_types (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  discount DECIMAL(5, 2) NOT NULL CHECK (discount BETWEEN 0 AND 1),
  description VARCHAR(250) NOT NULL
);

-- Ticket Statuses
CREATE TABLE ticket_statuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- Tickets Table
CREATE TABLE tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
  reservation_id UUID,
  price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
  type_id INT NOT NULL,
  status_id INT NOT NULL,
  CONSTRAINT fk_ticket_reservation_id FOREIGN KEY (reservation_id) REFERENCES reservations (id) ON DELETE CASCADE,
  CONSTRAINT fk_ticket_type FOREIGN KEY (type_id) REFERENCES ticket_types (id) ON DELETE CASCADE,
  CONSTRAINT fk_ticket_status FOREIGN KEY (status_id) REFERENCES ticket_statuses (id) ON DELETE CASCADE
);

-- Payment Statuses
CREATE TABLE payment_statuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- Payments Table
CREATE TABLE payment (
  id SERIAL PRIMARY KEY,
  order_id UUID NOT NULL,
  status_id INT NOT NULL,
  total_amount DECIMAL(10, 2) NOT NULL,
  payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_payment_reservation FOREIGN KEY (order_id) REFERENCES reservations (id) ON DELETE CASCADE,
  CONSTRAINT fk_payment_status FOREIGN KEY (status_id) REFERENCES payment_statuses (id) ON DELETE CASCADE
);

COMMENT ON TABLE users IS 'Stores user account information with role-based access';

COMMENT ON TABLE roles IS 'Defines user roles with different access levels';

COMMENT ON TABLE permissions IS 'Defines system-wide permissions for different actions';

COMMENT ON TABLE reservations IS 'Groups tickets into a single order for an event';

COMMENT ON TABLE tickets IS 'Represents individual tickets within group orders';

-- Initial values for Roles
INSERT INTO
  roles (name, description)
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

-- Initial values for permissions
INSERT INTO
  permissions (name, description)
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
  role_permissions (role_id, permission_id)
SELECT
  r.id,
  p.id
FROM
  roles r,
  permissions p
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
  reservation_statuses (name)
VALUES
  ('PENDING'),
  ('CONFIRMED'),
  ('CANCELLED');

-- Initial Ticket Statuses
INSERT INTO
  ticket_statuses (name)
VALUES
  ('AVAILABLE'),
  ('RESERVED'),
  ('SOLD'),
  ('CANCELLED');

-- Initial Payment Statuses
INSERT INTO
  payment_statuses (name)
VALUES
  ('PENDING'),
  ('COMPLETED'),
  ('FAILED'),
  ('REFUNDED');

-- Initial Ticket Types
INSERT INTO
  ticket_types (name, discount, description)
VALUES
  ('STANDARD', 0.00, 'Regular price ticket'),
  ('STUDENT', 0.20, '20% discount for students'),
  ('SENIOR', 0.15, '15% discount for seniors');
