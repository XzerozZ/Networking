package main

import (
	"log"
	"os"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type DistanceUpdate struct {
	Distances map[string]float64 `json:"distances"`
}

var (
	distances = map[string]float64{}
	mu        sync.Mutex
	nodeID    string
)

func updateHandler(c *fiber.Ctx) error {
	var req DistanceUpdate
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	mu.Lock()
	distances = req.Distances
	mu.Unlock()

	log.Printf("Node %s updated distances: %v", nodeID, distances)
	return c.JSON(fiber.Map{
		"status": "updated",
	})
}

func distancesHandler(c *fiber.Ctx) error {
	mu.Lock()
	defer mu.Unlock()
	return c.JSON(distances)
}

func healthHandler(c *fiber.Ctx) error {
	return c.SendString("OK")
}

func resetHandler(c *fiber.Ctx) error {
	mu.Lock()
	defer mu.Unlock()
	distances = map[string]float64{}
	return c.JSON(fiber.Map{"status": "reset completed"})
}

func main() {
	nodeID = os.Getenv("NODE_ID")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := fiber.New()

	app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,PUT,DELETE",
        AllowHeaders: "Content-Type",
    }))

	app.Post("/update", updateHandler)
	app.Get("/distances", distancesHandler)
	app.Get("/health", healthHandler)
	app.Post("/restart", resetHandler)

	log.Printf("Node %s service running on :%s", nodeID, port)
	log.Fatal(app.Listen(":" + port))
}
