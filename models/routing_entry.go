package models

type RoutingEntry struct {
	ID       uint   `gorm:"primaryKey"`
	RouterID uint   `gorm:"not null"`
	Dest     string `gorm:"not null"`
	Cost     int    `gorm:"not null"`
	NextHop  string `gorm:"not null"`
}
