package handler

import (
	"net/http"
	"os"

	"bhagirath-bhp/weather/handlers"
)

var metDir = os.Getenv("MET_DIR")

// Handler is the Vercel serverless entry-point for all /weather/* requests.
//
// vercel.json rewrites /weather/:path → /api/weather?path=:path so the
// dynamic segment arrives as the "path" query parameter.
//   - path == "current" (or absent) → serve latest available cycle
//   - anything else is treated as a YYYYMMDD date string
func Handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" || path == "current" {
		handlers.ServeCurrentWeather(metDir)(w, r)
		return
	}
	// Inject the date as a query param so ServeWeather can read it.
	q := r.URL.Query()
	q.Set("date", path)
	r.URL.RawQuery = q.Encode()
	handlers.ServeWeather(metDir)(w, r)
}
