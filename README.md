![ci](https://github.com/johndosdos/learn-cicd-starter/actions/workflows/ci.yml/badge.svg)
![cd](https://github.com/johndosdos/learn-cicd-starter/actions/workflows/cd.yml/badge.svg)

# Chatter

Chatter is a real-time chat application. Users can hop in and send messages that are instantly broadcast to all connected clients.

[Link to the app WIP]()

## Why I built this

I use messaging apps daily and never thought about how they work internally. So I made a real-time chat app to figure it out. I used WebSockets to handle the bulk of data transmission which include sending messages, typing indicators, and user presence count.

I kept the app simple, showing only user messages and usernames. No DMs, reactions or animations to focus on the logic behind real-time messaging.

## Tech stack

- **Go** for the backend.
- **WebSockets** for bidirectional client-server communication.
- **PostgreSQL** for persistence (using `sqlc` for type-safe queries and `goose` for migrations).
- **Docker** and **Docker Compose** for container orchestration.
- **HTMX** + **Templ** for the frontend (no framework).
- **Tailwind CSS** for styling.
- **VPS** for deployment.

## Security & Performance

- **Authentication**: Argon2id password hashing.
- **CSRF protection**: JWT + SameSite cookies.
- **XSS prevention**: bluemonday sanitization.
- **Rate limiting**: Per-user message throttling.
- **Connection pooling**: pgx for efficient DB access.

## Architecture Decisions

**Why WebSockets?** Bidirectional communication for typing indicators and message delivery.

**Why monolith?** Simpler deployment, easier debugging, and sufficient for current scale.

**Why PostgreSQL?** ACID guarantees for message persistence and mature ecosystem.

## How it works

Clients request a WebSocket connection upgrade through the `/chat ` endpoint –> messages are sent to the server via WebSockets –> saved to the database –> broadcast to all connected clients.

We keep track of connected clients using the in-memory map.

## Running it locally

During development, I used Docker and Compose to spin up and orchestrate the server, and DB containers. I've set up a Taskfile.yaml to run dev tasks. Feel free to take a look around!
