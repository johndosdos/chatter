FROM golang:1.25-alpine AS base
WORKDIR /goapp

RUN apk add --no-cache pnpm
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

RUN --mount=type=cache,target=/go/pkg/mod \
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
  go install github.com/pressly/goose/v3/cmd/goose@latest && \
  go install github.com/a-h/templ/cmd/templ@latest
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

FROM base AS dev
RUN --mount=type=cache,target=/go/pkg/mod \
  go install github.com/bokwoon95/wgo@latest

FROM base AS builder
COPY . .
RUN templ generate
RUN sqlc generate -f sqlc.yaml
RUN pnpm run build:css
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM scratch AS final
COPY --from=builder /server /server
COPY --from=builder /goapp/static /static
COPY --from=builder /goapp/sql /sql
COPY --from=builder /go/bin/goose /goose

EXPOSE 8080
ENTRYPOINT ["/server"]
