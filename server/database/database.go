package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var MG *MongoInstance

func Dbconn() error {

	// Database settings (insert your own database name and connection URI)
	const dbName = "book_worm"
	const mongoURI = "mongodb://mongo-db:27017/" + dbName

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	MG = &MongoInstance{
		Client: client,
		Db:     db}

	return nil
}
