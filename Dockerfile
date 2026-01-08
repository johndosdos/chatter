# base
FROM golang:1.25-alpine AS base
WORKDIR /goapp

RUN apk add --no-cache ca-certificates

RUN apk add --no-cache pnpm
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
  go install github.com/a-h/templ/cmd/templ@latest
COPY go.mod go.sum ./
RUN go mod download

# dev
FROM base AS dev
RUN --mount=type=cache,target=/go/pkg/mod \
  go install github.com/bokwoon95/wgo@latest

FROM base AS builder
COPY . .
RUN templ generate
RUN sqlc generate -f sqlc.yaml
RUN pnpm run build:css
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server .

# final
FROM scratch AS final
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /server /server
COPY --from=builder /goapp/static /static
COPY --from=builder /goapp/sql /sql

EXPOSE 8080
ENTRYPOINT ["/server"]
