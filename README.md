![ci](https://github.com/johndosdos/learn-cicd-starter/actions/workflows/ci.yml/badge.svg)
![cd](https://github.com/johndosdos/learn-cicd-starter/actions/workflows/cd.yml/badge.svg)

# Chatter

Chatter is a real-time chat application. Users can hop in and send messages that are instantly broadcast to all connected clients.

[Link to the app](https://chatter-server-678623746962.asia-southeast1.run.app)

## Why I built this

I use messaging apps daily and never thought about how they work internally. So I made a real-time chat app to figure it out. I started with WebSockets but switched to SSE because WebSockets were too complex for my use case. I added NATS JetStream as a message broker to ensure all server instances could handle messages to their respective clients.

I kept the app simple, showing only user messages and usernames. No DMs, reactions or animations to focus on the logic behind real-time messaging.

## Tech stack

- **Go** for the backend.
- **HTMX** + **Templ** for the frontend (no framework).
- **Tailwind CSS** for styling.
- **Server-Sent Events (SSE)** for pushing messages to clients.
- **PostgreSQL** for persistence (using `sqlc` for type-safe queries).
- **NATS JetStream** as the message broker.
- **Google Cloud Platform (GCP)** for building and deploying to the cloud.
- **Docker** and **Docker Compose** for local container orchestration.

## How it works

Clients request an open connection via GET through the `/stream` (SSE) endpoint —> messages are sent via POST —> saved to the database —> published to NATS —> pushed to all connected clients via SSE.

NATS handles the pub/sub part so multiple server instances could theoretically run without clients missing messages.

## Running it locally

During development, I used Docker and Compose to spin up and orchestrate the server, DB, and broker containers. I've set up a Taskfile.yaml to run dev tasks. Feel free to take a look around!
