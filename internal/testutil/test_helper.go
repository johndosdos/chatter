package testutil

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose"
)

func ProjectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "../../")
	return root
}

func DbInit() (*pgxpool.Pool, *sql.DB, string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	root := ProjectRoot()

	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil {
		log.Printf("failed to load .env file: %+v", err)
	}

	testURL := os.Getenv("TEST_DB_URL")
	if testURL == "" {
		log.Fatal("TEST_DB_URL environment variable is not set")
	}

	migDir := filepath.Join(root, "sql", "schema")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbPool, err := pgxpool.New(ctx, testURL)
	if err != nil {
		log.Fatalf("could not connect to the postgresql database: %v", err)
	}

	_ = goose.SetDialect("postgres")

	var dbErr error
	dbForGoose := stdlib.OpenDBFromPool(dbPool)
	if dbErr = goose.Reset(dbForGoose, migDir); dbErr != nil {
		dbForGoose.Close()
		log.Fatalf("goose.Reset() error = %+v", dbErr)
	}

	return dbPool, dbForGoose, migDir
}

func DbGooseUp(dbForGoose *sql.DB, migDir string) {
	if dbErr := goose.Up(dbForGoose, migDir); dbErr != nil {
		dbForGoose.Close()
		log.Fatalf("goose.Up() error = %+v", dbErr)
	}
}

func DbGooseReset(dbForGoose *sql.DB, migDir string) {
	if dbErr := goose.Reset(dbForGoose, migDir); dbErr != nil {
		dbForGoose.Close()
		log.Fatalf("goose.Up() error = %+v", dbErr)
	}
}

func DbCleanup(db *pgxpool.Pool, dir string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	dbForGoose := stdlib.OpenDBFromPool(db)
	DbGooseReset(dbForGoose, dir)

	if err := dbForGoose.Close(); err != nil {
		log.Fatalf("db.Close() error = %+v", err)
	}
}
