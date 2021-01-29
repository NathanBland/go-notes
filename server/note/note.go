package note

import (
	"github.com/NathanBland/go-notes/database"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type note struct {
	ID     string `json:"id,omitempty" bson:"_id,omitempty"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Rating int    `json:"rating"`
}

func Getnotes(c *fiber.Ctx) error {
	// get all records as a cursor
	query := bson.D{{}}
	cursor, err := database.MG.Db.Collection("notes").Find(c.Context(), query)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	var notes []note = make([]note, 0)

	if err := cursor.All(c.Context(), &notes); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(notes)
}

func Getnote(c *fiber.Ctx) error {
	id := c.Params("id")
	noteID, err := primitive.ObjectIDFromHex(id)
	collection := database.MG.Db.Collection("notes")

	if err != nil {
		return c.SendStatus(400)
	}

	query := bson.D{{Key: "_id", Value: noteID}}
	noteRecord := collection.FindOne(c.Context(), query)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return c.SendStatus(404)
		}
		return c.SendStatus(500)
	}
	note := &note{}
	noteRecord.Decode(note)

	return c.Status(200).JSON(note)
}
func Newnote(c *fiber.Ctx) error {
	collection := database.MG.Db.Collection("notes")

	// New note struct
	note := new(note)
	// Parse body into struct
	if err := c.BodyParser(note); err != nil {
		return c.Status(400).SendString(err.Error())
	}

	// force MongoDB to always set its own generated ObjectIDs
	note.ID = ""

	// insert the record
	insertionResult, err := collection.InsertOne(c.Context(), note)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	// get the just inserted record in order to return it as response
	filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
	createdRecord := collection.FindOne(c.Context(), filter)

	// decode the Mongo record into Employee
	creatednote := &note{}
	createdRecord.Decode(creatednote)

	// return the created Employee in JSON format
	return c.Status(201).JSON(creatednote)
}
func Deletenote(c *fiber.Ctx) error {
	noteID, err := primitive.ObjectIDFromHex(
		c.Params("id"),
	)

	// the provided ID might be invalid ObjectID
	if err != nil {
		return c.SendStatus(400)
	}

	// find and delete the note with the given ID
	query := bson.D{{Key: "_id", Value: noteID}}
	result, err := database.MG.Db.Collection("notes").DeleteOne(c.Context(), &query)

	if err != nil {
		return c.SendStatus(500)
	}

	// the note might not exist
	if result.DeletedCount < 1 {
		return c.SendStatus(404)
	}

	// the record was deleted
	return c.SendStatus(204)
}
