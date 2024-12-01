package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

type DistanceUpdate struct {
	Distances map[string]float64 `json:"distances"`
}

var distances = map[string]float64{}
var mu sync.Mutex
var nodeID string

func updateHandler(w http.ResponseWriter, r *http.Request) {
	var req DistanceUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	distances = req.Distances
	log.Printf("Node %s updated distances: %v", nodeID, distances)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func distancesHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(distances)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	nodeID = os.Getenv("NODE_ID")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/update", updateHandler)
	http.HandleFunc("/distances", distancesHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Node %s service running on :%s", nodeID, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
