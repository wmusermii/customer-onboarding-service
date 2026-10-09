package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"customer-onboarding-service/internal/adapter"
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

	txm := repository.NewTxManager(pool)
	caseRepo := repository.NewCaseRepo()
	eventLog := repository.NewEventLogRepo()

	caseSvc := service.NewCaseService(
		txm,
		repository.NewCustomerRepo(),
		caseRepo,
		eventLog,
		repository.NewOutboxRepo(),
	)

	verifSvc := service.NewVerificationService(
		pool, txm, caseRepo,
		repository.NewVerificationRepo(),
		repository.NewDocumentRepo(),
		eventLog,
		adapter.NewDummyDocStore(),
		adapter.NewDummyEKYC(cfg.EKYCBaseURL, cfg.EKYCAPIKey),
	)

	srv := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: handler.NewRouter(pool,
			handler.NewCaseHandler(caseSvc),
			handler.NewVerificationHandler(verifSvc),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on :%s", cfg.HTTPPort)
	log.Fatal(srv.ListenAndServe())
}
