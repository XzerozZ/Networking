package database

import (
	"log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"project/models"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("routers.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// AutoMigrate Models
	err = DB.AutoMigrate(&models.Router{}, &models.Neighbor{}, &models.RoutingEntry{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database connected and migrated successfully!")
}
