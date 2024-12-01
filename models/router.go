package models

type Router struct {
	ID           uint           `gorm:"primaryKey"`
	Port         int            `gorm:"unique;not null"`
	Neighbors    []Neighbor     `gorm:"foreignKey:RouterID"`
	RoutingTable []RoutingEntry `gorm:"foreignKey:RouterID"`
}
