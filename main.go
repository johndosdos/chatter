// Package main our entry point.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

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
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var err error
	dbURL := os.Getenv("DB_URL")
	dbConn, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Printf("main: cannot connect to postgresql database: %v", err)
		return
	}
	dbQueries = database.New(dbConn)

	// hub.Run is our central hub that is always listening for client related
	// events.
	hub := ws.NewHub()
	go hub.Run(ctx, dbQueries)

	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/account/login", handler.ServeLogin(dbQueries))
	http.Handle("/account/signup", handler.ServeSignup(dbQueries))

	// Load chat history on HTTP GET on initial connection before starting websockets.
	// This is to prevent issues regarding resending chat history on websocket reconnection.
	http.Handle("/messages", internal.Middleware(handler.ServeMessages(dbQueries), dbQueries))
	http.Handle("/ws", internal.Middleware(handler.ServeWs(hub, dbQueries), dbQueries))
	http.Handle("/chat", internal.Middleware(handler.ServeChat(), dbQueries))

	http.Handle("/", handler.ServeRoot())

	defer dbConn.Close()

	log.Println("Server starting at port", server.Addr)
	log.Fatal(server.ListenAndServe())
}
