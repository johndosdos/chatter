package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgxpool"
	components "github.com/johndosdos/chatter/components/chat"
	"github.com/johndosdos/chatter/internal/database"

	"github.com/johndosdos/chatter/internal/handler"
	ws "github.com/johndosdos/chatter/internal/websocket"
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
	http.Handle("/", templ.Handler(components.Base()))
	http.HandleFunc("/ws", handler.ServeWs(ctx, hub, dbQueries))

	defer dbConn.Close()

	log.Println("Server starting at port", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
