// https://www.digitalocean.com/community/tutorials/how-to-use-go-with-mongodb-using-the-mongodb-go-driver
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/NathanBland/go-vite-docker-starter/book"
	"github.com/NathanBland/go-vite-docker-starter/database"
	"github.com/gofiber/fiber"
    "github.com/jinzhu/gorm"
    // _ "github.com/jinzhu/gorm/dialects/sqlite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var ctx = context.TODO()

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://mongo-db:27017/")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("noted").Collection("notes")
}

func helloWorld(c *fiber.Ctx) {
	c.Send("hello world")
}

func setupRoutes(app *fiber.App) {
	app.Get("/api/v1/book", book.GetBooks)
	app.Get("/api/v1/book/:id", book.GetBook)
	app.Post("/api/v1/book", book.NewBook)
	app.Delete("/api/v1/book/:id", book.DeleteBook)
}

func initDatabase() {
	var err error
	database.DBConn, err = gorm.Open("sqlite3", "books.db")
	if err != nil {
		log.Fatal("Failed to connect to database")
	}
	fmt.Println("Database connection successfully established")

	database.DBConn.AutoMigrate(&book.Book{})
	fmt.Println("Database Migrated")
}

func main() {
	app := fiber.New()

	initDatabase()
	defer database.DBConn.Close()

	setupRoutes(app)

	log.Fatal(app.Listen(":3001"))
}
