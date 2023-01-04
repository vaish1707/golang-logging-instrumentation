package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/vaish1707/golang-logging-instrumentation/config"
	"github.com/vaish1707/golang-logging-instrumentation/datastore"
	"github.com/vaish1707/golang-logging-instrumentation/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "order-service"

var (
	mongodbClient *datastore.MongoClientCfg
	srv           *http.Server
	orderUrl      string
	userUrl       string
	tracer        trace.Tracer
)

func setupServer() {
	router := mux.NewRouter()
	router.HandleFunc("/orders", otelhttp.NewHandler(createOrder(), "CreateOrder").ServeHTTP).Methods(http.MethodPost)
	router.Use(utils.LoggingMW)
	router.Use(utils.LogRequestID)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost},
	})

	srv = &http.Server{
		Addr:    orderUrl,
		Handler: c.Handler(router),
	}

	log.Printf("Order service running at: %s", orderUrl)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("failed to setup http server: %v", err)
	}
}

func initDB() {
	var err error
	mongodbClient, err = datastore.NewClient()
	if err != nil {
		fmt.Println("Error while connecting to mongodb", err)
	}
}

func main() {
	// read the config from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file", err)
	}
	orderUrl = os.Getenv("ORDER_URL")
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
	tracer = otel.Tracer(serviceName)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	initDB()
	go setupServer()

	<-sigint
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server shutdown failed")
	}
}
