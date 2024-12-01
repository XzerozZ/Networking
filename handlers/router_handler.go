package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"project/database"
	"project/models"
	"project/services"
)

func StartRouterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Port      int             `json:"port"`
		Neighbors []models.Neighbor `json:"neighbors"`
	}

	// Decode request body
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate port
	if request.Port <= 0 || request.Port > 65535 {
		http.Error(w, "Port must be a valid number between 1 and 65535", http.StatusBadRequest)
		return
	}

	// Create router
	router := models.Router{
		Port:         request.Port,
		Neighbors:    request.Neighbors,
		RoutingTable: []models.RoutingEntry{},
	}

	// Initialize routing table
	for _, neighbor := range request.Neighbors {
		router.RoutingTable = append(router.RoutingTable, models.RoutingEntry{
			Dest:    neighbor.Address,
			Cost:    neighbor.Cost,
			NextHop: neighbor.Address,
		})
	}

	// Save to database
	database.DB.Create(&router)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Router started on port %d", request.Port),
	})
}

func StopRouterHandler(w http.ResponseWriter, r *http.Request) {
	port := r.URL.Query().Get("port")
	if port == "" {
		http.Error(w, "Port is required", http.StatusBadRequest)
		return
	}

	var router models.Router
	result := database.DB.Where("port = ?", port).First(&router)
	if result.RowsAffected == 0 {
		http.Error(w, "Router not found", http.StatusNotFound)
		return
	}

	database.DB.Delete(&router)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Router on port %s stopped", port),
	})
}

func ConnectRouterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Port1 int `json:"port1"`
		Port2 int `json:"port2"`
		Cost  int `json:"cost"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var router1, router2 models.Router
	if database.DB.Where("port = ?", request.Port1).First(&router1).RowsAffected == 0 ||
		database.DB.Where("port = ?", request.Port2).First(&router2).RowsAffected == 0 {
		http.Error(w, "One or both routers not found", http.StatusNotFound)
		return
	}

	// Add neighbors
	router1.Neighbors = append(router1.Neighbors, models.Neighbor{
		RouterID: router1.ID,
		Address:  fmt.Sprintf("localhost:%d", request.Port2),
		Cost:     request.Cost,
	})

	router2.Neighbors = append(router2.Neighbors, models.Neighbor{
		RouterID: router2.ID,
		Address:  fmt.Sprintf("localhost:%d", request.Port1),
		Cost:     request.Cost,
	})

	// Save updates
	database.DB.Save(&router1)
	database.DB.Save(&router2)

	// Trigger routing updates
	services.BellmanFordUpdate(&router1)
	services.BellmanFordUpdate(&router2)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Routers connected successfully",
	})
}
