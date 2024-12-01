package main

import (
	"log"
	"net/http"
	"project/database"
	"project/handlers"
)

func main() {
	// เชื่อมต่อฐานข้อมูล
	database.InitDB()

	// กำหนด HTTP Handlers
	http.HandleFunc("/start", handlers.StartRouterHandler)
	http.HandleFunc("/connect", handlers.ConnectRouterHandler)
	http.HandleFunc("/routes/", handlers.GetRoutesHandler)
	http.HandleFunc("/stop/", handlers.StopRouterHandler)

	// เริ่มเซิร์ฟเวอร์
	log.Println("Server starting on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
