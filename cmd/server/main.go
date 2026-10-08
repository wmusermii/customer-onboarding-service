package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"customer-onboarding-service/internal/config"
	"customer-onboarding-service/internal/handler"
	"customer-onboarding-service/internal/platform/postgres"
	"customer-onboarding-service/internal/repository"
	"customer-onboarding-service/internal/service"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	log.Println("database connected")

	caseSvc := service.NewCaseService(
		repository.NewTxManager(pool),
		repository.NewCustomerRepo(),
		repository.NewCaseRepo(),
		repository.NewEventLogRepo(),
		repository.NewOutboxRepo(),
	)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler.NewRouter(pool, handler.NewCaseHandler(caseSvc)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on :%s", cfg.HTTPPort)
	log.Fatal(srv.ListenAndServe())
}
