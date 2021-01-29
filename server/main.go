// https://www.digitalocean.com/community/tutorials/how-to-use-go-with-mongodb-using-the-mongodb-go-driver
package main

import (
	"log"
	"github.com/NathanBland/go-vite-docker-starter/book"
	"github.com/NathanBland/go-vite-docker-starter/database"
	"github.com/gofiber/fiber/v2"
)

func setupRoutes(app *fiber.App) {
	app.Get("/api/v1/book", book.GetBooks)
	app.Get("/api/v1/book/:id", book.GetBook)
	app.Post("/api/v1/book", book.NewBook)
	app.Delete("/api/v1/book/:id", book.DeleteBook)
}

func main() {
	app := fiber.New()
	// Connect to the database
	if err := database.Dbconn(); err != nil {
		log.Fatal(err)
	}

	setupRoutes(app)

	log.Fatal(app.Listen(":3001"))
}
