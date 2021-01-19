package book

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/NathanBland/go-vite-docker-starter/database"
)

type Book struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Title string `json:"title"`
	Author string `json:"author"`
	Rating int `json:"rating"`
}


func GetBooks(c *fiber.Ctx) error {
	// get all records as a cursor
	query := bson.D{{}}
	cursor, err := database.MG.Db.Collection("books").Find(c.Context(), query)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	var books []Book = make([]Book, 0)

	if err := cursor.All(c.Context(), &books); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(books)
}

func GetBook(c *fiber.Ctx) error {
	id := c.Params("id")
	bookID, err := primitive.ObjectIDFromHex(id)
	collection := database.MG.Db.Collection("books")

	if err != nil {
		return c.SendStatus(400)
	}

	query := bson.D{{Key: "_id", Value: bookID}}
	bookRecord := collection.FindOne(c.Context(), query)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return c.SendStatus(404)
		}
		return c.SendStatus(500)
	}
	book := &Book{}
	bookRecord.Decode(book)
	
	return c.Status(200).JSON(book)
}
func NewBook(c *fiber.Ctx) error {
	collection := database.MG.Db.Collection("books")

		// New Book struct
		book := new(Book)
		// Parse body into struct
		if err := c.BodyParser(book); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		// force MongoDB to always set its own generated ObjectIDs
		book.ID = ""

		// insert the record
		insertionResult, err := collection.InsertOne(c.Context(), book)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// get the just inserted record in order to return it as response
		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)

		// decode the Mongo record into Employee
		createdBook := &Book{}
		createdRecord.Decode(createdBook)

		// return the created Employee in JSON format
		return c.Status(201).JSON(createdBook)
}
func DeleteBook(c *fiber.Ctx) error {
	bookID, err := primitive.ObjectIDFromHex(
		c.Params("id"),
	)

	// the provided ID might be invalid ObjectID
	if err != nil {
		return c.SendStatus(400)
	}

	// find and delete the book with the given ID
	query := bson.D{{Key: "_id", Value: bookID}}
	result, err := database.MG.Db.Collection("books").DeleteOne(c.Context(), &query)

	if err != nil {
		return c.SendStatus(500)
	}

	// the book might not exist
	if result.DeletedCount < 1 {
		return c.SendStatus(404)
	}

	// the record was deleted
	return c.SendStatus(204)
}
