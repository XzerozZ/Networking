package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	"node2": 1e9, // Initial value as infinity
	"node3": 1e9,
}

var nodes = []string{"node1", "node2", "node3"}
var mu sync.Mutex

func updateHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Relaxation
	if distances[req.Source]+req.Weight < distances[req.Dest] {
		distances[req.Dest] = distances[req.Source] + req.Weight
		propagateUpdates()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(distances)
}

func distancesHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(distances)
}

func propagateUpdates() {
	for _, node := range nodes {
		url := fmt.Sprintf("http://%s:8080/update", node)
		data, _ := json.Marshal(DistanceUpdate{Distances: distances})

		go func(nodeURL string, payload []byte) {
			for retries := 0; retries < 3; retries++ { // Retry up to 3 times
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

func finalResultHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"final_distances": distances,
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	http.HandleFunc("/update", updateHandler)
	http.HandleFunc("/distances", distancesHandler)
	http.HandleFunc("/final_result", finalResultHandler)
	http.HandleFunc("/health", healthHandler)

	log.Println("Controller service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
