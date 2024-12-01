package models

type Neighbor struct {
	ID       uint   `gorm:"primaryKey"`
	RouterID uint   `gorm:"not null"`
	Address  string `gorm:"unique;not null"`
	Cost     int    `gorm:"not null"`
}
