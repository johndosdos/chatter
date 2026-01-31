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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/johndosdos/chatter/internal"
	"github.com/johndosdos/chatter/internal/database"
	"github.com/johndosdos/chatter/internal/handler"
	ws "github.com/johndosdos/chatter/internal/websocket"
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

	/* 	// Init NATS
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

	   	js, err := jetstream.New(conn)
	   	if err != nil {
	   		log.Fatalf("failed to create jetstream instance: %v", err)
	   	}

	   	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
	   		Name:     broker.StreamName,
	   		Subjects: []string{broker.SubjectGlobalRoom},
	   		MaxBytes: 1 << 30, // 1GB max storage
	   	})
	   	if err != nil {
	   		log.Fatalf("failed to create/update stream: %v", err)
	   	} */

	// Init DB
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	dbConn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("could not connect to the postgresql database: %v", err)
	}

	dbQueries := database.New(dbConn)

	// hub.Run is our central hub that is always listening for client related events.
	hub := ws.NewHub(dbQueries)
	go hub.Run(ctx)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	fs := http.FileServer(http.Dir("static"))
	r.Handle("/static*", http.StripPrefix("/static", fs))
	r.Get("/", handler.ServeRoot())

	r.Route("/account", func(r chi.Router) {
		r.Get("/login", handler.ServeLoginPage())
		r.Post("/login", handler.SubmitLoginForm(dbQueries))

		r.Get("/signup", handler.ServeSignupPage())
		r.Post("/signup", handler.SubmitSignupForm(dbQueries))

		r.Post("/logout", handler.SubmitLogoutReq(dbQueries))
	})

	r.Group(func(r chi.Router) {
		r.Use(internal.Middleware(dbQueries))
		r.Get("/messages", handler.ServeMessages(dbQueries))
		r.Get("/ws", handler.ServeWs(hub, dbQueries))
		r.Get("/chat", handler.ServeChat())
	})

	server := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		log.Printf("Server starting at 0.0.0.0:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("Shutdown signal received; shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println(err)
	}

	/* 	// Drain NATS connection.
	   	if err := conn.Drain(); err != nil {
	   		log.Printf("couldn't drain NATS conn: %+v", err)
	   	} */

	// Close DB connection.
	dbConn.Close()

	log.Println("Server stopped")
}
