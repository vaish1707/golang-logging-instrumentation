package datastore

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

type MongoClientCfg struct {
	MongoClient *mongo.Client
}

func NewClient() (*MongoClientCfg, error) {
	// read the config from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file", err)
	}

	var (
		mongo_url = os.Getenv("MONGO_DB_URL")
	)

	// open up our database connection.
	client, err := mongo.NewClient(options.Client().ApplyURI(mongo_url).SetMonitor(otelmongo.NewMonitor()))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Mongodb client Started")

	return &MongoClientCfg{
		MongoClient: client,
	}, nil
}
