package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Neighbor struct {
	Address string `json:"address"`
	Cost    int    `json:"cost"`
}

type RoutingEntry struct {
	Cost    int    `json:"cost"`
	NextHop string `json:"next_hop"`
}

type Router struct {
	Port         int
	Neighbors    map[string]int
	RoutingTable map[string]RoutingEntry
	Lock         sync.Mutex
	StopChan     chan bool
}

func NewRouter(port int, neighbors []Neighbor) (*Router, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d. Port must be between 1 and 65535", port)
	}

	neighborMap := make(map[string]int)
	for _, neighbor := range neighbors {
		parts := strings.Split(neighbor.Address, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid neighbor address format: %s. Must be in 'host:port' format", neighbor.Address)
		}

		neighborPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid neighbor port in address %s: %v", neighbor.Address, err)
		}

		if neighborPort == port {
			return nil, fmt.Errorf("cannot add neighbor with same port as router: %d", port)
		}

		if neighbor.Cost < 0 {
			return nil, fmt.Errorf("invalid neighbor cost: %d. Cost must be non-negative", neighbor.Cost)
		}

		neighborMap[neighbor.Address] = neighbor.Cost
	}

	routingTable := make(map[string]RoutingEntry)
	selfAddress := fmt.Sprintf("localhost:%d", port)
	routingTable[selfAddress] = RoutingEntry{Cost: 0, NextHop: selfAddress}

	for address, cost := range neighborMap {
		routingTable[address] = RoutingEntry{Cost: cost, NextHop: address}
	}

	return &Router{
		Port:         port,
		Neighbors:    neighborMap,
		RoutingTable: routingTable,
		StopChan:     make(chan bool),
	}, nil
}

func (r *Router) BellmanFordUpdate() bool {
	updated := false
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for destination, route := range r.RoutingTable {
		for neighbor, linkCost := range r.Neighbors {
			if neighborRoute, ok := r.RoutingTable[neighbor]; ok {
				newCost := linkCost + neighborRoute.Cost
				if newCost < route.Cost || 
				   (newCost == route.Cost && strings.Compare(neighbor, route.NextHop) < 0) {
					r.RoutingTable[destination] = RoutingEntry{
						Cost:    newCost, 
						NextHop: neighbor,
					}
					updated = true
				}
			}
		}
	}
	return updated
}

func (r *Router) BroadcastRoutingTable() {
	r.Lock.Lock()
	data, err := json.Marshal(r.RoutingTable)
	r.Lock.Unlock()
	if err != nil {
		log.Printf("Router %d: Error marshalling routing table: %v", r.Port, err)
		return
	}

	for neighbor := range r.Neighbors {
		parts := strings.Split(neighbor, ":")
		if len(parts) != 2 {
			log.Printf("Router %d: Invalid neighbor address: %s", r.Port, neighbor)
			continue
		}
		conn, err := net.Dial("udp", neighbor)
		if err != nil {
			log.Printf("Router %d: Error broadcasting to %s: %v", r.Port, neighbor, err)
			continue
		}
		conn.Write(data)
		conn.Close()
	}
}

func (r *Router) ReceiveUpdates() {
	address := fmt.Sprintf("localhost:%d", r.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Printf("Router %d: Error resolving address %s: %v", r.Port, address, err)
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("Router %d: Error starting UDP listener on %s: %v", r.Port, address, err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 4096)
	for {
		select {
		case <-r.StopChan:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			var receivedTable map[string]RoutingEntry
			err = json.Unmarshal(buffer[:n], &receivedTable)
			if err != nil {
				log.Printf("Router %d: Error unmarshalling received data: %v", r.Port, err)
				continue
			}

			updated := false
			r.Lock.Lock()
			for destination, info := range receivedTable {
				if existing, ok := r.RoutingTable[destination]; !ok || info.Cost < existing.Cost {
					r.RoutingTable[destination] = info
					updated = true
				}
			}
			r.Lock.Unlock()

			if updated {
				r.BroadcastRoutingTable()
			}
		}
	}
}

func (r *Router) Start() {
	go r.ReceiveUpdates()

	go func() {
		for {
			select {
			case <-r.StopChan:
				return
			default:
				if r.BellmanFordUpdate() {
					r.BroadcastRoutingTable()
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func (r *Router) Stop() {
	close(r.StopChan)
}

var routers = make(map[int]*Router)

func startRouterHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.New(log.Writer(), "START ROUTER: ", log.Ldate|log.Ltime|log.Lshortfile)

	if r.Method != http.MethodPost {
		logger.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Port      int        `json:"port"`
		Neighbors []Neighbor `json:"neighbors"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&request)
	if err != nil {
		logger.Printf("Failed to decode request body: %v", err)

		if err, ok := err.(*json.UnmarshalTypeError); ok {
			logger.Printf("Type error: Field %v, Type %v", err.Field, err.Type)
		}

		if request.Port == 0 {
			logger.Printf("Port is zero or not provided")
		}

		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if request.Port == 0 {
		logger.Printf("Port cannot be zero")
		http.Error(w, "Port must be a non-zero positive integer", http.StatusBadRequest)
		return
	}

	logger.Printf("Received router start request for port %d with %d neighbors", 
		request.Port, len(request.Neighbors))

	if _, exists := routers[request.Port]; exists {
		logger.Printf("Router already exists on port %d", request.Port)
		http.Error(w, "Router already exists on this port", http.StatusBadRequest)
		return
	}

	router, err := NewRouter(request.Port, request.Neighbors)
	if err != nil {
		logger.Printf("Failed to create router: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	routers[request.Port] = router
	router.Start()

	logger.Printf("Router successfully started on port %d", request.Port)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Router started on port %d", request.Port),
	})
}

func getRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portStr := strings.TrimPrefix(r.URL.Path, "/routes/")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "Invalid port", http.StatusBadRequest)
		return
	}

	router, exists := routers[port]
	if !exists {
		http.Error(w, "Router not found", http.StatusNotFound)
		return
	}

	router.Lock.Lock()
	defer router.Lock.Unlock()
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]map[string]RoutingEntry{
		"routing_table": router.RoutingTable,
	})
}

func stopRouterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portStr := strings.TrimPrefix(r.URL.Path, "/stop/")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "Invalid port", http.StatusBadRequest)
		return
	}

	router, exists := routers[port]
	if !exists {
		http.Error(w, "Router not found", http.StatusNotFound)
		return
	}

	router.Stop()
	delete(routers, port)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Router on port %d stopped", port),
	})
}

func connectRouterHandler(w http.ResponseWriter, r *http.Request) {
    logger := log.New(log.Writer(), "CONNECT ROUTER: ", log.Ldate|log.Ltime|log.Lshortfile)

    if r.Method != http.MethodPost {
        logger.Printf("Method not allowed: %s", r.Method)
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var request struct {
        Port1 int `json:"port1"`
        Port2 int `json:"port2"`
        Cost  int `json:"cost"`
    }

    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&request); err != nil {
        logger.Printf("Failed to decode request body: %v", err)
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    // Validate ports
    if request.Port1 <= 0 || request.Port2 <= 0 {
        logger.Printf("Invalid ports: port1=%d, port2=%d", request.Port1, request.Port2)
        http.Error(w, "Ports must be positive integers", http.StatusBadRequest)
        return
    }

    // Validate cost
    if request.Cost < 0 {
        logger.Printf("Invalid cost: %d", request.Cost)
        http.Error(w, "Cost must be non-negative", http.StatusBadRequest)
        return
    }

    // Check if both routers exist
    router1, router1Exists := routers[request.Port1]
    router2, router2Exists := routers[request.Port2]
    
    if !router1Exists {
        logger.Printf("Router on port %d not found", request.Port1)
        http.Error(w, fmt.Sprintf("Router on port %d not found. Please start the router first.", request.Port1), http.StatusNotFound)
        return
    }

    if !router2Exists {
        logger.Printf("Router on port %d not found", request.Port2)
        http.Error(w, fmt.Sprintf("Router on port %d not found. Please start the router first.", request.Port2), http.StatusNotFound)
        return
    }

    // Create router addresses
    addr1 := fmt.Sprintf("localhost:%d", request.Port1)
    addr2 := fmt.Sprintf("localhost:%d", request.Port2)

    // Add bidirectional connection
    router1.Lock.Lock()
    router1.Neighbors[addr2] = request.Cost
    router1.RoutingTable[addr2] = RoutingEntry{
        Cost:    request.Cost,
        NextHop: addr2,
    }
    router1.Lock.Unlock()

    router2.Lock.Lock()
    router2.Neighbors[addr1] = request.Cost
    router2.RoutingTable[addr1] = RoutingEntry{
        Cost:    request.Cost,
        NextHop: addr1,
    }
    router2.Lock.Unlock()

    // Trigger routing updates
    router1.BellmanFordUpdate()
    router1.BroadcastRoutingTable()
    router2.BellmanFordUpdate()
    router2.BroadcastRoutingTable()

    logger.Printf("Connected router on port %d to %d with cost %d", 
        request.Port1, request.Port2, request.Cost)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "message": "Routers connected successfully",
    })
}

func main() {
	log.SetPrefix("ROUTER APP: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	http.HandleFunc("/start", startRouterHandler)
	http.HandleFunc("/connect", connectRouterHandler)
	http.HandleFunc("/routes/", getRoutesHandler)
	http.HandleFunc("/stop/", stopRouterHandler)

	log.Println("Server starting on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}