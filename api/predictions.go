package api

import "net/http"

func Predictions(w http.ResponseWriter, r *http.Request) {
	serve(w, r, "/api/predictions")
}
