package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type ServiceRegistry struct {
	Services map[string]string
	lock     sync.RWMutex
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		Services: make(map[string]string),
	}
}

func (sr *ServiceRegistry) Register(serviceName, address string) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	sr.Services[serviceName] = address
}

func (sr *ServiceRegistry) Discover(serviceName string) (string, bool) {
	sr.lock.RLock()
	defer sr.lock.RUnlock()
	address, exists := sr.Services[serviceName]
	return address, exists
}

func main() {
	registry := NewServiceRegistry()
	
	r := http.NewServeMux()

	r.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ServiceName string `json:"service_name"`
			Address    string `json:"address"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		registry.Register(req.ServiceName, req.Address)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Service registered successfully",
		})
	})

	r.HandleFunc("/discover", func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.URL.Query().Get("service")
		if serviceName == "" {
			http.Error(w, "Service name is required", http.StatusBadRequest)
			return
		}

		address, exists := registry.Discover(serviceName)
		if !exists {
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"address": address,
		})
	})

	log.Println("Discovery Service starting on :8083")
	log.Fatal(http.ListenAndServe(":8083", r))
}