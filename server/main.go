// https://www.digitalocean.com/community/tutorials/how-to-use-go-with-mongodb-using-the-mongodb-go-driver
package main

import (
	"context"
	"log"
	"time"

	"github.com/NathanBland/go-vite-docker-starter/book"
	"github.com/NathanBland/go-vite-docker-starter/database"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var ctx = context.TODO()

// func init() {
// 	clientOptions := options.Client().ApplyURI("mongodb://mongo-db:27017/")
// 	client, err := mongo.Connect(ctx, clientOptions)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	err = client.Ping(ctx, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	collection = client.Database("noted").Collection("notes")
// }

func setupRoutes(app *fiber.App) {
	app.Get("/api/v1/book", book.GetBooks)
	app.Get("/api/v1/book/:id", book.GetBook)
	app.Post("/api/v1/book", book.NewBook)
	app.Delete("/api/v1/book/:id", book.DeleteBook)
}

func initDatabase() error {
	
	type MongoInstance struct {
		Client *mongo.Client
		Db     *mongo.Database
	}
	
	
	// Database settings (insert your own database name and connection URI)
	const dbName = "fiber_test"
	const mongoURI = "mongodb://mongo-db:27017/" + dbName

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	database.MG = &MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil
}

func main() {
	app := fiber.New()
	// Connect to the database
	if err := initDatabase(); err != nil {
		log.Fatal(err)
	}

	setupRoutes(app)

	log.Fatal(app.Listen(":3001"))
}
