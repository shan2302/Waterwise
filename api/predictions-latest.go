package api

import "net/http"

func PredictionsLatest(w http.ResponseWriter, r *http.Request) {
	serve(w, r, "/api/predictions/latest")
}
