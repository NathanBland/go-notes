// https://www.digitalocean.com/community/tutorials/how-to-use-go-with-mongodb-using-the-mongodb-go-driver
package main

import (
	"log"
	"github.com/NathanBland/go-notes/note"
	"github.com/NathanBland/go-notes/database"
	"github.com/gofiber/fiber/v2"
)

func setupRoutes(app *fiber.App) {
	app.Get("/api/v1/note", note.Getnotes)
	app.Get("/api/v1/note/:id", note.Getnote)
	app.Post("/api/v1/note", note.Newnote)
	app.Delete("/api/v1/note/:id", note.Deletenote)
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
