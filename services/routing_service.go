package services

import (
	"project/models"
	"sync"
)

// RoutingService provides the core logic for managing routing tables and updates.
type RoutingService struct {
	mutex sync.Mutex // Ensures thread-safe access to routing tables.
}

// UpdateRoutingTable performs the Bellman-Ford algorithm to update the routing table of the given router.
// It returns true if the routing table was updated, otherwise false.
func (s *RoutingService) UpdateRoutingTable(router *models.Router) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	updated := false
	routingTable := router.RoutingTable
	neighbors := router.Neighbors

	for _, neighbor := range neighbors {
		neighborAddress := neighbor.Address
		neighborCost := neighbor.Cost

		for _, entry := range routingTable {
			newCost := neighborCost + entry.Cost

			// Check if we found a shorter or equal-cost path with a lexicographically smaller next hop.
			if existing, ok := routingTable[entry.Dest]; !ok || newCost < existing.Cost || 
				(newCost == existing.Cost && neighborAddress < existing.NextHop) {

				routingTable[entry.Dest] = models.RoutingEntry{
					Dest:    entry.Dest,
					Cost:    newCost,
					NextHop: neighborAddress,
				}
				updated = true
			}
		}

		// Ensure neighbor itself is in the routing table
		if _, ok := routingTable[neighborAddress]; !ok {
			routingTable[neighborAddress] = models.RoutingEntry{
				Dest:    neighborAddress,
				Cost:    neighborCost,
				NextHop: neighborAddress,
			}
			updated = true
		}
	}

	return updated
}

// BroadcastRoutingTable simulates sending routing table updates to neighbors.
// In a real application, this would involve network operations.
func (s *RoutingService) BroadcastRoutingTable(router *models.Router) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Placeholder: Replace this with actual network broadcasting logic.
	for _, entry := range router.RoutingTable {
		// Example: Log the routing table for each neighbor
		println("Broadcasting -> Dest:", entry.Dest, "Cost:", entry.Cost, "Next Hop:", entry.NextHop)
	}
}
