package actions

import (
	// "bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gobuffalo/buffalo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

// PhotosHandler handles GET requests to /{username}/photos
func PhotosHandler(c buffalo.Context) error {
	// Get the username from the URL parameters
	username := c.Param("username")
	// fmt.Println("Username:", username)

	if username == "" {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": "Username is required"}))
	}

	uri := os.Getenv("MONGOLINK")
	fmt.Println("Uri:", uri)
	if uri == "" {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "MongoDB connection string is not set"}))
	}

	if client == nil {
		var err error
		clientOptions := options.Client().ApplyURI(uri).
			SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
			SetConnectTimeout(60 * time.Second).
			SetMaxPoolSize(10)

		client, err = mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "Failed to connect to MongoDB"}))
		}

		// Verify the connection
		if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
			return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "Failed to ping MongoDB"}))
		}
	}

	collection := client.Database("mydb").Collection("Posts")
	var user struct {
		DisplayPicture string `bson:"DisplayPicture"`
	}

	err := collection.FindOne(context.TODO(), map[string]string{"Username": username}).Decode(&user)
	if err != nil {
		return c.Render(http.StatusNotFound, r.JSON(map[string]string{"error": "User not found"}))
	}

	imageURL := user.DisplayPicture
	if !startsWith(imageURL, "https://") {
		imageURL = "https://s3.amazonaws.com/profile-display-images/" + user.DisplayPicture
	}

	imageResponse, err := http.Get(imageURL)
	if err != nil || imageResponse.StatusCode != http.StatusOK {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "Failed to fetch image"}))
	}
	defer imageResponse.Body.Close()

	imageBuffer, err := io.ReadAll(imageResponse.Body)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "Failed to read image response"}))
	}

	contentType := imageResponse.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	_, err = c.Response().Write(imageBuffer)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "Failed to write image response"}))
	}

	return nil
}

// startsWith is a helper function to check if a string starts with a given prefix
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
