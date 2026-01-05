// Package main our entry point.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/johndosdos/chatter/internal"
	"github.com/johndosdos/chatter/internal/broker"
	"github.com/johndosdos/chatter/internal/chat"
	"github.com/johndosdos/chatter/internal/database"

	"github.com/johndosdos/chatter/internal/handler"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Init NATS
	var natsCredentials []nats.Option
	natsURL := os.Getenv("NATS_URL")

	if cred := os.Getenv("NATS_CRED"); cred != "" {
		natsCredentials = append(natsCredentials, nats.UserCredentials(cred))
	} else {
		natsCredentials = append(natsCredentials, nats.UserInfo(os.Getenv("NATS_USER"), os.Getenv("NATS_PASSWORD")))
	}

	conn, err := nats.Connect(natsURL, natsCredentials...)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer conn.Drain()

	js, err := jetstream.New(conn)
	if err != nil {
		log.Fatalf("%v", err)
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     broker.StreamName,
		Subjects: []string{broker.SubjectGlobalRoom},
	})
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Init DB
	dbURL := os.Getenv("DB_URL")
	dbConn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("could not connect to the postgresql database: %v", err)
	}
	dbQueries := database.New(dbConn)

	// hub.Run is our central hub that is always listening for client related
	// events.
	hub := chat.NewHub(js, dbQueries)
	go hub.Run(ctx, stream)

	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/account/login", handler.ServeLogin(dbQueries))
	http.Handle("/account/signup", handler.ServeSignup(dbQueries))
	http.Handle("/account/logout", handler.ServeLogout(dbQueries))

	// Load chat history on HTTP GET on initial connection before starting websockets.
	// This is to prevent issues regarding resending chat history on websocket reconnection.
	http.Handle("/messages", internal.Middleware(handler.ServeMessages(dbQueries), dbQueries))
	http.Handle("/stream", internal.Middleware(handler.StreamSSE(hub, dbQueries), dbQueries))
	http.Handle("/chat", internal.Middleware(handler.ServeChat(), dbQueries))
	http.Handle("/send", internal.Middleware(chat.Send(hub, dbQueries), dbQueries))

	http.Handle("/", handler.ServeRoot())

	go func(server *http.Server) {
		log.Println("Server starting at port", server.Addr)
		log.Fatal(server.ListenAndServe())
	}(server)

	<-ctx.Done()
	log.Printf("Shutdown signal received; shutting down...")
}
