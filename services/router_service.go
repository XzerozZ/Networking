package services

import (
	"project/models"
)

// BellmanFordUpdate performs the Bellman-Ford algorithm to update routing tables.
// Returns a boolean indicating if any changes were made to the routing table.
func BellmanFordUpdate(router *models.Router) bool {
	updated := false

	// Map to hold the routing table for quick access
	table := make(map[string]models.RoutingEntry)
	for _, entry := range router.RoutingTable {
		table[entry.Dest] = entry
	}

	// Iterate over each neighbor and update the routing table
	for _, neighbor := range router.Neighbors {
		neighborCost := neighbor.Cost
		for _, entry := range table {
			newCost := neighborCost + entry.Cost

			// Check if the new cost is better (lower) than the existing one
			if current, exists := table[entry.Dest]; !exists || newCost < current.Cost {
				table[entry.Dest] = models.RoutingEntry{
					Dest:    entry.Dest,
					Cost:    newCost,
					NextHop: neighbor.Address,
				}
				updated = true
			}
		}

		// Ensure the neighbor itself is part of the routing table
		if _, exists := table[neighbor.Address]; !exists {
			table[neighbor.Address] = models.RoutingEntry{
				Dest:    neighbor.Address,
				Cost:    neighborCost,
				NextHop: neighbor.Address,
			}
			updated = true
		}
	}

	// Convert the updated map back into the router's routing table slice
	router.RoutingTable = make([]models.RoutingEntry, 0, len(table))
	for _, entry := range table {
		router.RoutingTable = append(router.RoutingTable, entry)
	}

	return updated
}

// BroadcastRoutingTable simulates broadcasting the routing table to all neighbors.
// In a real implementation, this would involve network communication.
func BroadcastRoutingTable(router *models.Router) {
	// Placeholder: Print the routing table or log it to simulate broadcasting.
	for _, entry := range router.RoutingTable {
		// Example: Print destination, cost, and next hop
		println("Destination:", entry.Dest, "Cost:", entry.Cost, "Next Hop:", entry.NextHop)
	}
}
