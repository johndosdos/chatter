# Chatter

Chatter is a simple, real-time chat application built with **Go**, **HTMX**, and **WebSockets**. Users can hop in and send messages that are instantly broadcast to all connected clients.  

The app is deployed on `Render` and uses `Neon`, a serverless PostgreSQL database for message persistence.

## Live Demo
Access the live app [`here`](https://chat-app-wgpp.onrender.com).  

When visiting, you’ll be prompted to enter a username. After that, you’ll join the chatroom and can start sending messages immediately.

## Features
- **Real-Time Messaging**: Messages are sent and received instantly without refreshing the page, powered by WebSockets.
- **Persistent Chat History**: Messages are stored in PostgreSQL and loaded when a new user joins.
- **Simple User Interface**: Clean, minimal UI built with Tailwind CSS.

## How It Works
The application uses a modern stack for real-time communication.

1. **Frontend**:
   Built with the Go package `a-h/templ` for server-side templating and `htmx` for dynamic UI updates. When a user sends a message, htmx sends the content through a WebSocket connection.

2. **Backend**:
   A Go server manages WebSocket connections. Each client is registered with a central *hub*.

3. **WebSocket Communication**:
   The hub broadcasts messages to all connected clients, sanitizes them to prevent XSS attacks, and saves them to the Neon database.

4. **Database**:
   A serverless PostgreSQL instance on Neon stores all messages, providing persistent chat history.

## Technologies Used
- **Backend**: [`Go`](https://go.dev/)  
- **Frontend**: [`templ`](https://github.com/a-h/templ), [`htmx`](https://htmx.org/), [`Tailwind CSS`](https://tailwindcss.com/)  
- **Real-Time Communication**: WebSockets ([`gorilla/websocket`](https://github.com/gorilla/websocket))  
- **Database**: PostgreSQL with [`sqlc`](https://sqlc.dev/) for type-safe query generation  
- **Deployment**:  
  - Hosted on [`Render`](https://render.com/)  
  - Database on [`Neon`](https://neon.com/)
