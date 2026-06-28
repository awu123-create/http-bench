package main

import (
	"context"
	"log"
	"os"

	"github.com/awu123-create/http-bench/internal/api"
	"github.com/awu123-create/http-bench/internal/storage"
	storagedb "github.com/awu123-create/http-bench/internal/storage/db"
	"github.com/awu123-create/http-bench/internal/task"
)

func main() {
	addr := getenv("HTTP_BENCH_ADDR", ":8080")
	dbPath := getenv("HTTP_BENCH_DB", "data/http-bench.db")
	migrationDir := getenv("HTTP_BENCH_MIGRATIONS", "migrations")

	sqlDB, err := storage.Open(context.Background(), dbPath, migrationDir)
	if err != nil {
		log.Fatalf("open sqlite database: %v", err)
	}
	defer sqlDB.Close()

	queries := storagedb.New(sqlDB)
	manager := task.NewManager(queries)
	router := api.NewRouter(manager)
	defer manager.Close()

	if err := router.Run(addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
