package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5/pgxpool"

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
	http.Handle("/", http.RedirectHandler("/account/login", http.StatusSeeOther))
	http.Handle("/account/login", handler.ServeLogin(ctx, dbQueries))
	http.Handle("/account/signup", handler.ServeSignup(ctx, dbQueries))

	// Load chat history on HTTP GET on initial connection before starting websockets.
	// This is to prevent issues regarding resending chat history on websocket reconnection.
	http.Handle("/messages", handler.ServeMessages(ctx, dbQueries))
	http.Handle("/ws", handler.ServeWs(ctx, hub, dbQueries))

	http.Handle("/chat", handler.ServeChat(ctx))

	defer dbConn.Close()

	log.Println("Server starting at port", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
