#!/bin/bash
set -e

### nocodb.sh
# creates a separate user and database for nocodb metadata
###

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	CREATE USER $NC_USER WITH PASSWORD '$NC_PASSWORD';
	CREATE DATABASE $NC_NAME;
  GRANT ALL PRIVILEGES ON DATABASE $NC_NAME TO $NC_USER;
  ALTER DATABASE $NC_NAME OWNER TO $NC_USER
EOSQL
