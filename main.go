package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/johndosdos/chatter/internal"
	"github.com/johndosdos/chatter/internal/database"
	ws "github.com/johndosdos/chatter/internal/websocket"

	"github.com/johndosdos/chatter/internal/handler"
)

var (
	dbConn    *pgxpool.Pool
	dbQueries *database.Queries
)

func main() {
	port := ":8080"
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// database init start
	var err error
	dbURL := os.Getenv("DB_URL")
	dbConn, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Printf("[Error] cannot connect to postgresql database: %v", err)
		return
	}

	dbQueries = database.New(dbConn)
	// database init end

	// client hub init start
	// hub.Run is our central hub that is always listening for client related
	// events.
	hub := ws.NewHub()
	go hub.Run(ctx, dbQueries)
	// client hub init end

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/account/login", handler.ServeLogin(dbQueries))
	http.Handle("/account/signup", handler.ServeSignup(dbQueries))

	// Load chat history on HTTP GET on initial connection before starting websockets.
	// This is to prevent issues regarding resending chat history on websocket reconnection.
	http.Handle("/messages", internal.Middleware(handler.ServeMessages(dbQueries)))
	http.Handle("/ws", internal.Middleware(handler.ServeWs(hub, dbQueries)))
	http.Handle("/chat", internal.Middleware(handler.ServeChat()))

	http.Handle("/api/token/refresh", handler.RefreshToken(dbQueries))

	http.Handle("/", handler.ServeRoot())

	defer dbConn.Close()

	log.Println("Server starting at port", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
