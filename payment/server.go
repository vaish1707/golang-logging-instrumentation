package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/vaish1707/golang-logging-instrumentation/config"
	"github.com/vaish1707/golang-logging-instrumentation/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const serviceName = "payment-service"

var (
	srv        *http.Server
	paymentUrl string
	userUrl    string
)

func setupServer() {
	router := mux.NewRouter()
	router.HandleFunc("/payments/transfer/id/{userID}", otelhttp.NewHandler(transferAmount(), "transferamount").ServeHTTP).Methods(http.MethodPut, http.MethodOptions)
	router.Use(utils.LoggingMW)
	router.Use(utils.LogRequestID)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost},
	})

	srv = &http.Server{
		Addr:    paymentUrl,
		Handler: c.Handler(router),
	}

	log.Printf("Payment service running at: %s", paymentUrl)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("failed to setup http server: %v", err)
	}
}

func main() {
	// read the config from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file", err)
	}
	paymentUrl = os.Getenv("PAYMENT_URL")
	userUrl = os.Getenv("USER_URL")

	// setup tracer
	tp, err := config.Init(serviceName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	go setupServer()

	<-sigint
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server shutdown failed")
	}
}
