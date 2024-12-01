package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"project/database"
	"project/models"
)

func GetRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portStr := r.URL.Path[len("/routes/"):]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "Invalid port", http.StatusBadRequest)
		return
	}

	var router models.Router
	result := database.DB.Preload("RoutingTable").Where("port = ?", port).First(&router)
	if result.RowsAffected == 0 {
		http.Error(w, "Router not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(router.RoutingTable)
}
