# get the golang
FROM golang:1.21-alpine

# set working directory
WORKDIR /api

# copy the module files
COPY go.mod go.sum ./
RUN go mod download

# copy the rest of the application
COPY . .

# build the application
RUN go build -o main .


EXPOSE 8080
CMD ["./main"]
