package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"bhagirath-bhp/weather/handlers"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	metDir := "/home/loki/useful/projects/weather/_ignore_/metfiles/"
	http.HandleFunc("/weather/{date}", handlers.ServeWeather(metDir))
	http.HandleFunc("/weather/current", handlers.ServeCurrentWeather(metDir))
	http.Handle("/", http.FileServer(http.Dir("./web")))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}

}
