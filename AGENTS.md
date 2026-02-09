# AGENTS.md

Guidelines for agentic coding agents working in this Go/Templ/HTMX real-time chat application.

## Technology Stack

- **Language:** Go 1.25.1
- **Router:** chi/v5
- **Frontend:** HTMX + Templ (type-safe templates)
- **Styling:** Tailwind CSS
- **Real-time:** WebSockets (coder/websocket)
- **Messaging:** NATS JetStream (broker layer)
- **Database:** PostgreSQL (pgx/v5 + sqlc)
- **Auth:** JWT + Argon2id password hashing
- **Sanitization:** bluemonday

---

## Build & Development Commands

### Core Workflow
```bash
# Full dev mode (CSS + Templates + Server with hot reload)
task dev

# Run server only with hot reload
task local::server

# Build for production
go build -o ./bin/chatter main.go
```

### Testing Commands
```bash
# Run all tests
go test -v ./...

# Run tests in a specific package
go test -v ./internal/auth/

# Run a single test by name
go test -v -run TestHashPassword ./internal/auth/

# Run tests with race detector (ALWAYS before committing concurrent code)
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Code Generation
```bash
# Generate SQLC database code (after modifying sql/queries/ or sql/schema/)
sqlc generate

# Generate Templ templates (after modifying .templ files)
templ generate

# Watch and regenerate templates in dev
wgo -file .templ -xfile ._templ.go templ generate
```

### Code Quality
```bash
# Run linter
golangci-lint run

# Format all Go code
go fmt ./...

# Security vulnerability scan
govulncheck ./...
```

### Pre-Commit Verification Checklist
Run these before committing:
```bash
go fmt ./... && \
golangci-lint run && \
go test -race ./... && \
templ generate && \
sqlc generate
```

### CSS Build
```bash
# Build Tailwind CSS once
pnpm run build:css

# Watch CSS for changes
pnpm run watch:css
```

### Database Operations
```bash
# Run migrations
task db::up

# Reset database
task db::reset

# Using goose directly
goose up
goose reset
```

### Docker Operations
```bash
# Start services
task compose::up

# Start with rebuild
task compose::up-build

# Stop services
task compose::down
```

---

## Code Style Guidelines

### Import Organization
Organize in three groups with blank lines:
1. Standard library
2. Third-party
3. Internal (`github.com/johndosdos/chatter/internal/*`)

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgtype"

    "github.com/johndosdos/chatter/internal/auth"
    "github.com/johndosdos/chatter/internal/database"
)
```

### Naming Conventions

**Functions:**
- Public: `PascalCase` (`HashPassword`, `ServeWs`)
- Private: `camelCase` (`setTokens`, `parseRequest`)
- Handlers: `ServeX()` or `SubmitX()` (`ServeLoginPage`, `SubmitSignupForm`)
- Factories: `NewX()` (`NewHub`, `NewClient`)

**Variables:**
- Local: `camelCase` (`ctx`, `db`, `userID`)
- Constants: `PascalCase` (idiomatic Go, NOT UPPER_SNAKE_CASE)
- Context keys: `PascalCase` with `Key` suffix (`UserIDKey`)
- HTTP params: `w` (writer), `r` (request)

**Types:**
- Structs: `PascalCase` (`Hub`, `Client`, `ChatMessage`)
- Interfaces: Descriptive verbs or `-er` suffix (`Sanitizer`, `MessageStore`)

---

## Error Handling Patterns

### Wrap Errors with Context
Always wrap errors with package context using `%w`:

```go
// Good: provides call chain context
return "", fmt.Errorf("auth.HashPassword: failed to generate hash: %w", err)

// Check wrapped errors
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrUserNotFound
}
```

### Structured Logging (slog)
Use Go 1.21+ structured logging for observability:

```go
import "log/slog"

// Good: structured with context
slog.Error("failed to upgrade connection",
    "error", err,
    "client_ip", r.RemoteAddr,
    "user_id", userID,
)

// Good: info with structured fields
slog.Info("user authenticated",
    "user_id", userID,
    "method", "jwt",
)

// Avoid: unstructured log.Printf (legacy only)
log.Printf("failed: %v", err)
```

### Error Response Patterns
```go
// Handler error responses
func respondError(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    http.Error(w, msg, code)
}

// Use appropriate status codes
// 400: Bad Request (malformed input)
// 401: Unauthorized (missing/invalid auth)
// 403: Forbidden (valid auth, no permission)
// 404: Not Found
// 422: Unprocessable Entity (valid syntax, semantic error)
// 500: Internal Server Error (log details, return generic message)
```

---

## Testing Patterns

### Table-Driven Tests
Use table-driven tests for comprehensive coverage:

```go
func TestHashPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid password", "securePass123!", false},
        {"empty password", "", true},
        {"unicode password", "password123", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := HashPassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Organization
- Test files: `*_test.go` in same package
- Test function names: `TestFunctionName_Scenario`
- Use `t.Helper()` for test helper functions
- Use `t.Parallel()` for independent tests

### Mocking Strategy
Prefer interfaces over mocking libraries:

```go
// Define interface for dependencies
type MessageStore interface {
    GetMessages(ctx context.Context, limit int32) ([]database.Message, error)
}

// In tests, create simple mock implementations
type mockMessageStore struct {
    messages []database.Message
    err      error
}

func (m *mockMessageStore) GetMessages(ctx context.Context, limit int32) ([]database.Message, error) {
    return m.messages, m.err
}
```

---

## HTTP API Conventions

### Status Codes
| Code | Usage |
|------|-------|
| 200 | Success (GET, POST returning data) |
| 201 | Created (POST creating resource) |
| 204 | No Content (DELETE, logout) |
| 400 | Bad Request (malformed JSON, missing fields) |
| 401 | Unauthorized (missing/invalid token) |
| 403 | Forbidden (valid auth, insufficient permissions) |
| 404 | Not Found |
| 422 | Unprocessable Entity (validation failed) |
| 500 | Internal Server Error |

### Handler Pattern
```go
func ServeMessages(db *database.Queries) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // 1. Parse input
        limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
        if err != nil {
            http.Error(w, "invalid limit parameter", http.StatusBadRequest)
            return
        }

        // 2. Call business logic
        messages, err := db.GetMessages(ctx, int32(limit))
        if err != nil {
            slog.Error("failed to get messages", "error", err)
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }

        // 3. Render response
        component := chat.MessageList(messages)
        component.Render(ctx, w)
    }
}
```

---

## Database Patterns

### Query Conventions (SQLC)
- All database access through `internal/database/` (generated by sqlc)
- Never write raw SQL in handlers
- Use context for cancellation: `db.GetUser(ctx, userID)`

### Transaction Pattern
```go
func (s *Service) TransferFunds(ctx context.Context, from, to string, amount int) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx) // No-op if committed

    qtx := s.queries.WithTx(tx)

    if err := qtx.DeductBalance(ctx, from, amount); err != nil {
        return fmt.Errorf("deduct: %w", err)
    }

    if err := qtx.AddBalance(ctx, to, amount); err != nil {
        return fmt.Errorf("add: %w", err)
    }

    return tx.Commit(ctx)
}
```

### Migration Conventions
- One migration per logical change
- Always include rollback (`-- +goose Down`)
- Test migrations on fresh database before committing

---

## Concurrency Patterns

### Goroutine Lifecycle
Always ensure goroutines can be stopped:

```go
func (h *Hub) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            // Cleanup and exit
            return
        case reg := <-h.Register:
            h.clients[reg.Client] = true
        case client := <-h.Unregister:
            delete(h.clients, client)
        case msg := <-h.ClientMsg:
            h.broadcast(msg)
        }
    }
}
```

### Channel Best Practices
- Prefer buffered channels for decoupling producer/consumer
- Close channels from the sender side only
- Use `context.Context` for cancellation, not channel closing

### WebSocket Connection Pattern
```go
type Hub struct {
    Register   chan Registration
    Unregister chan *Client
    ClientMsg  chan model.ChatMessage
    clients    map[*Client]bool
    mu         sync.RWMutex // Protect map access
}
```

---

## Security Guidelines

### Input Sanitization
Always sanitize user input with bluemonday:

```go
sanitized := h.sanitizer.Sanitize(payload.Content)
payload.Content = sanitized
```

### Authentication Patterns
- JWT with refresh token pattern
- HTTP-only cookies for tokens (prevents XSS token theft)
- Argon2id for password hashing
- Validate token on every protected request

### WebSocket Security
- Validate origins on upgrade
- Use structured message validation
- Implement connection timeouts
- Rate limit message sending

### SQL Injection Prevention
- Always use parameterized queries (sqlc handles this)
- Never concatenate user input into SQL strings

### Secrets Management
- Never commit `.env` files with real credentials
- Use environment variables for all secrets
- Validate required env vars at startup with clear error messages

---

## Architecture & Package Structure

```
internal/
├── auth/              # JWT & password hashing (auth.go, auth_test.go)
├── broker/            # NATS message brokering (pubsub.go, routing.go)
├── database/          # SQLC-generated queries (DO NOT EDIT MANUALLY)
├── handler/           # HTTP handlers (ServeX, SubmitX patterns)
├── model/             # Domain models (message.go)
├── middleware.go      # Auth middleware (single file, not a package)
└── websocket/         # WebSocket hub & clients (hub.go, client.go, server.go)

components/            # Templ templates (type-safe HTML)
├── auth/
├── base.templ
└── chat/

sql/
├── queries/           # SQLC query definitions (.sql files)
└── schema/            # Database migrations (goose format)

static/                # Static assets (CSS, JS, images)
```

### Dependency Injection
- Pass dependencies explicitly through constructors
- Use interfaces for testability
- Avoid global state

```go
// Good: explicit dependencies
func NewHandler(db *database.Queries, hub *websocket.Hub) *Handler {
    return &Handler{db: db, hub: hub}
}

// Avoid: global variables
var globalDB *database.Queries // Don't do this
```

### Context Propagation
- Pass `context.Context` as first parameter
- Use context for request-scoped values (user ID, request ID)
- Use context for cancellation in long-running operations

```go
func (h *Handler) ServeMessages(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := ctx.Value(UserIDKey).(string)
    // ...
}
```

---

## Observability

### Health Checks
Implement health endpoints for container orchestration:

```go
r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    // Check critical dependencies
    if err := db.Ping(r.Context()); err != nil {
        http.Error(w, "database unhealthy", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
})
```

### Request Logging
Use chi middleware for structured request logs:

```go
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
```

---

## Common Pitfalls to Avoid

1. **Goroutine leaks**: Always provide cancellation mechanism
2. **Nil pointer panics**: Check interface values before type assertion
3. **Race conditions**: Use `-race` flag in tests, protect shared state
4. **Unbounded growth**: Set limits on channels, slices, maps
5. **Silent failures**: Always handle errors, never use `_` for errors
6. **Premature optimization**: Profile before optimizing
7. **Over-abstraction**: Keep it simple, add abstraction when needed
