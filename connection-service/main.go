package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Maintain global router registry
var (
	routers     = make(map[int]*Router)
	routersLock sync.Mutex
)

func main() {
	log.SetPrefix("CONNECTION SERVICE: ")
	r := http.NewServeMux()

	r.HandleFunc("/connect", connectRouterHandler)

	log.Println("Connection Service starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", r))
}

func connectRouterHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Port1 int `json:"port1"`
		Port2 int `json:"port2"`
		Cost  int `json:"cost"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input
	if request.Port1 <= 0 || request.Port2 <= 0 || request.Cost < 0 {
		http.Error(w, "Invalid ports or cost", http.StatusBadRequest)
		return
	}

	// Simulate connection logic
	log.Printf("Connecting routers on ports %d and %d with cost %d", 
		request.Port1, request.Port2, request.Cost)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Routers connected successfully",
	})
}