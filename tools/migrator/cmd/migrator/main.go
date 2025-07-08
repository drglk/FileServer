package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	log := setupLogger()

	user, password, addr, port, dbname := mustLoadEnv()

	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, addr, port, dbname)
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Error("migrate new", "err", err)
		os.Exit(1)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Error("migrate up", "err", err)
		os.Exit(1)
	}

	version, dirty, err := m.Version()
	if err != nil {
		log.Error("migrate version", "err", err)
		os.Exit(1)
	}

	log.Info("migrations applied successfully", "version", version, "dirty", dirty)

}

func mustLoadEnv() (user string, password string, addr string, port int, dbname string) {
	user = os.Getenv("POSTGRES_USER")
	if user == "" {
		panic("POSTGRES_USER is required")
	}

	password = os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		panic("POSTGRES_USER is required")
	}

	addr = os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		panic("POSTGRES_PASSWORD is required")
	}

	portStr := os.Getenv("POSTGRES_PORT")
	if portStr == "" {
		panic("POSTGRES_PORT is required")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(err)
	}

	dbname = os.Getenv("POSTGRES_DB")
	if dbname == "" {
		panic("POSTGRES_DB is required")
	}

	return
}

func setupLogger() *slog.Logger {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	return log
}
