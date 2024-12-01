package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ServiceRegistry struct {
	RouterServices map[int]string
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		RouterServices: make(map[int]string),
	}
}

func (sr *ServiceRegistry) RegisterRouterService(port int, url string) {
	sr.RouterServices[port] = url
}

func (sr *ServiceRegistry) GetRouterServiceURL(port int) (string, bool) {
	url, exists := sr.RouterServices[port]
	return url, exists
}

func main() {
	registry := NewServiceRegistry()
	r := mux.NewRouter()

	// Start Router Endpoint
	r.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Port      int        `json:"port"`
			Neighbors []Neighbor `json:"neighbors"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Forward request to router-service
		routerServiceURL := "http://router-service:8081/start"
		jsonReq, _ := json.Marshal(req)
		resp, err := http.Post(routerServiceURL, "application/json", bytes.NewBuffer(jsonReq))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to start router: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Register router service
		registry.RegisterRouterService(req.Port, routerServiceURL)

		// Forward response
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Router started successfully",
		})
	}).Methods("POST")

	// Connect Router Endpoint
	r.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Port1 int `json:"port1"`
			Port2 int `json:"port2"`
			Cost  int `json:"cost"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Forward request to connection-service
		connectionServiceURL := "http://connection-service:8082/connect"
		jsonReq, _ := json.Marshal(req)
		resp, err := http.Post(connectionServiceURL, "application/json", bytes.NewBuffer(jsonReq))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to connect routers: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Forward response
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Routers connected successfully",
		})
	}).Methods("POST")

	// Get Routes Endpoint
	r.HandleFunc("/routes/{port}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		port, err := strconv.Atoi(vars["port"])
		if err != nil {
			http.Error(w, "Invalid port", http.StatusBadRequest)
			return
		}

		// Forward request to router-service
		routerServiceURL := "http://router-service:8081/routes"
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%d", routerServiceURL, port), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get routes: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Forward response
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"routing_table": resp.Body,
		})
	}).Methods("GET")

	// Stop Router Endpoint
	r.HandleFunc("/stop/{port}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		port, err := strconv.Atoi(vars["port"])
		if err != nil {
			http.Error(w, "Invalid port", http.StatusBadRequest)
			return
		}

		// Forward request to router-service
		routerServiceURL := "http://router-service:8081/stop"
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/%d", routerServiceURL, port), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop router: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Forward response
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Router stopped successfully",
		})
	}).Methods("POST")

	log.Println("API Gateway starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Neighbor type for request decoding
type Neighbor struct {
	Address string `json:"address"`
	Cost    int    `json:"cost"`
}