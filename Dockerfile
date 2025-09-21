# Build the backend.
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install sqlc CLI for schemas
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
# Install goose CLI for migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./

# Run sqlc
RUN sqlc generate -f ./sqlc.yaml

RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

# Create the final image.
FROM scratch
COPY --from=builder ./server ./server
COPY --from=builder ./app/static ./static
COPY --from=builder ./go/bin/goose ./goose
COPY --from=builder ./.env.production .

# Install dotenvx for injecting production database string
RUN wget -qO- https://dotenvx.sh | sh

EXPOSE 8080
CMD [ "dotenvx", "run", "-f", ".env.production", "--", "/server" ]