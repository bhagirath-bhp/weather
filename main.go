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
	metDir := os.Getenv("MET_DIR")
	if metDir == "" {
		metDir = "./_ignore_/metfiles/"
	}
	http.HandleFunc("/weather/{date}", handlers.ServeWeather(metDir))
	http.HandleFunc("/weather/current", handlers.ServeCurrentWeather(metDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "./index.html")
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}

}
