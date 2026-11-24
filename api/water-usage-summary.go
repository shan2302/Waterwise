package api

import "net/http"

func WaterUsageSummary(w http.ResponseWriter, r *http.Request) {
	serve(w, r, "/api/water-usage/summary")
}
