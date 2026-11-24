package api

import (
	"net/http"
)

func WaterUsage(w http.ResponseWriter, r *http.Request) {
	serve(w, r, "/api/water-usage")
}
