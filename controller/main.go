package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type UpdateRequest struct {
	Source string  `json:"source"`
	Dest   string  `json:"dest"`
	Weight float64 `json:"weight"`
}

type DistanceUpdate struct {
	Distances map[string]float64 `json:"distances"`
}

var distances = map[string]float64{
	"node1": 0,
	"node2": 1e9,
	"node3": 1e9,
}

var nodes = []string{"node1", "node2", "node3"}
var mu sync.Mutex

func updateHandler(c *fiber.Ctx) error {
	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	mu.Lock()
	if distances[req.Source]+req.Weight < distances[req.Dest] {
		distances[req.Dest] = distances[req.Source] + req.Weight
		propagateUpdates()
	}
	mu.Unlock()

	return c.JSON(distances)
}

func distancesHandler(c *fiber.Ctx) error {
	mu.Lock()
	defer mu.Unlock()
	return c.JSON(distances)
}

func propagateUpdates() {
	for _, node := range nodes {
		url := fmt.Sprintf("http://%s:8080/update", node)
		data, _ := json.Marshal(DistanceUpdate{Distances: distances})

		go func(nodeURL string, payload []byte) {
			for retries := 0; retries < 3; retries++ {
				resp, err := http.Post(nodeURL, "application/json", bytes.NewReader(payload))
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					return
				}
				log.Printf("Retry %d for Node %s failed: %v", retries+1, nodeURL, err)
			}
			log.Printf("Failed to update node %s after retries", nodeURL)
		}(url, data)
	}
}

func finalResultHandler(c *fiber.Ctx) error {
	mu.Lock()
	defer mu.Unlock()
	return c.JSON(fiber.Map{
		"final_distances": distances,
	})
}

func healthHandler(c *fiber.Ctx) error {
	return c.SendString("OK")
}
func resetHandler(c *fiber.Ctx) error {
	mu.Lock()
	defer mu.Unlock()
	distances = map[string]float64{
		"node1": 0,
		"node2": 1e9,
		"node3": 1e9,
	}
	propagateReset()

	return c.JSON(fiber.Map{"status": "reset completed"})
}
func propagateReset() {
	for _, node := range nodes {
		url := fmt.Sprintf("http://%s:8080/restart", node)
		data, _ := json.Marshal(DistanceUpdate{Distances: distances})

		go func(nodeURL string, payload []byte) {
			for retries := 0; retries < 3; retries++ {
				resp, err := http.Post(nodeURL, "application/json", bytes.NewReader(payload))
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					return
				}
				log.Printf("Retry %d for Node %s failed: %v", retries+1, nodeURL, err)
			}
			log.Printf("Failed to update node %s after retries", nodeURL)
		}(url, data)
	}
}

func main() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,PUT,DELETE",
        AllowHeaders: "Content-Type",
    }))
	
	app.Post("/update", updateHandler)
	app.Get("/distances", distancesHandler)
	app.Get("/final_result", finalResultHandler)
	app.Get("/health", healthHandler)
	app.Post("/restart", resetHandler)

	log.Println("Controller service running on :8080")
	log.Fatal(app.Listen(":8080"))
}
