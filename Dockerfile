# get the golang
FROM golang

# set working directory
WORKDIR /usr/src/event-api

# copy the module files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# copy the rest of the application
COPY . .

# build the application
RUN go build -v -o /usr/local/bin/event-api .

EXPOSE 8080
CMD ["event-api", "--populate"]
