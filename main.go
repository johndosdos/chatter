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
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/johndosdos/chatter/internal"
	"github.com/johndosdos/chatter/internal/broker"
	"github.com/johndosdos/chatter/internal/chat"
	"github.com/johndosdos/chatter/internal/database"

	"github.com/johndosdos/chatter/internal/handler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %+v", err)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Init server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              "0.0.0.0:" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
	}

	go func() {
		log.Printf("Server starting at 0.0.0.0:%s", port)
		log.Fatal(server.ListenAndServe())
	}()

	// Init NATS
	log.Println("Starting application...")
	log.Println("Initializing NATS connection...")

	var natsCredentials []nats.Option

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		log.Fatal("NATS_URL environment variable is not set")
	}

	if cred := os.Getenv("NATS_CRED"); cred != "" {
		natsCredentials = append(natsCredentials, nats.UserCredentials(cred))
	} else if user, pass := os.Getenv("NATS_USER"), os.Getenv("NATS_PASSWORD"); user != "" && pass != "" {
		natsCredentials = append(natsCredentials, nats.UserInfo(user, pass))
	}

	natsCredentials = append(natsCredentials, nats.Timeout(5*time.Second))

	conn, err := nats.Connect(natsURL, natsCredentials...)
	if err != nil {
		log.Fatalf("failed to connect to nats: %v", err)
	}
	defer conn.Drain() //nolint:errcheck

	js, err := jetstream.New(conn)
	if err != nil {
		log.Fatalf("failed to create jetstream instance: %v", err)
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     broker.StreamName,
		Subjects: []string{broker.SubjectGlobalRoom},
		MaxBytes: 1 << 30,
	})
	if err != nil {
		log.Fatalf("failed to create/update stream: %v", err)
	}

	// Init DB
	log.Println("Initializing Database connection...")

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	dbConn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("could not connect to the postgresql database: %v", err)
	}

	dbQueries := database.New(dbConn)

	// Init Hub
	hub := chat.NewHub(js, dbQueries)
	go hub.Run(ctx, stream)

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.Handle("/account/login", handler.ServeLogin(dbQueries))
	mux.Handle("/account/signup", handler.ServeSignup(dbQueries))
	mux.Handle("/account/logout", handler.ServeLogout(dbQueries))

	// Load chat history on HTTP GET on initial connection before starting websockets.
	// This is to prevent issues regarding resending chat history on websocket reconnection.
	mux.Handle("/messages", internal.Middleware(handler.ServeMessages(dbQueries), dbQueries))
	mux.Handle("/stream", internal.Middleware(handler.StreamSSE(hub, dbQueries), dbQueries))
	mux.Handle("/chat", internal.Middleware(handler.ServeChat(), dbQueries))
	mux.Handle("/send", internal.Middleware(chat.Send(hub, dbQueries), dbQueries))

	mux.Handle("/", handler.ServeRoot())

	server.Handler = mux

	<-ctx.Done()
	log.Printf("Shutdown signal received; shutting down...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println(err)
	}

	log.Println("Server stopped")
}
