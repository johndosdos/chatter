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
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/johndosdos/chatter/internal"
	"github.com/johndosdos/chatter/internal/broker"
	"github.com/johndosdos/chatter/internal/broker/worker"
	"github.com/johndosdos/chatter/internal/database"
	ws "github.com/johndosdos/chatter/internal/websocket"

	"github.com/johndosdos/chatter/internal/handler"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Init NATS
	natsURL := os.Getenv("NATS_URL")

	conn, err := nats.Connect(natsURL, nats.UserInfo(os.Getenv("NATS_USER"), os.Getenv("NATS_PASSWORD")))
	if err != nil {
		log.Fatalf("main: %v", err)
	}
	defer conn.Drain()

	js, err := jetstream.New(conn)
	if err != nil {
		log.Fatalf("main: %v", err)
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     broker.StreamName,
		Subjects: []string{broker.SubjectGlobalRoom},
	})
	if err != nil {
		log.Fatalf("main: %v", err)
	}

	// Init DB
	dbURL := os.Getenv("DB_URL")
	dbConn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("main: could not connect to the postgresql database: %v", err)
	}
	dbQueries := database.New(dbConn)

	// hub.Run is our central hub that is always listening for client related
	// events.
	hub := ws.NewHub(js, dbQueries)
	go hub.Run(ctx)

	err = broker.Subscriber(ctx, stream, worker.WorkerHub(hub))
	if err != nil {
		log.Printf("main: could not connect to the postgresql database: %v", err)
	}

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
	http.Handle("/account/logout", handler.ServeLogout(dbQueries))

	// Load chat history on HTTP GET on initial connection before starting websockets.
	// This is to prevent issues regarding resending chat history on websocket reconnection.
	http.Handle("/messages", internal.Middleware(handler.ServeMessages(dbQueries), dbQueries))
	http.Handle("/ws", internal.Middleware(handler.ServeWs(hub, dbQueries), dbQueries))
	http.Handle("/chat", internal.Middleware(handler.ServeChat(), dbQueries))

	http.Handle("/", handler.ServeRoot())

	log.Println("Server starting at port", server.Addr)
	log.Fatal(server.ListenAndServe())
}
