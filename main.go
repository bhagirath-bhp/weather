package main

import (
	"log"
	"net/http"

	"bhagirath-bhp/weather/handlers"
)

func main() {
	metDir := "/home/loki/useful/projects/weather/_ignore_/metfiles/"
	http.HandleFunc("/weather/{date}", handlers.ServeWeather(metDir))
	http.HandleFunc("/weather/current", handlers.ServeCurrentWeather(metDir))
	http.Handle("/", http.FileServer(http.Dir("./web")))
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
