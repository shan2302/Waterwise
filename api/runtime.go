package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"water-demand/backend"
)

var (
	initializeLock sync.Mutex
	apiHandler     http.Handler
)

func serve(w http.ResponseWriter, r *http.Request, path string) {
	if err := initialize(); err != nil {
		log.Printf("initialize API function: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Database unavailable. Configure a reachable PostgreSQL database for this deployment.",
		}); err != nil {
			log.Printf("write initialization error response: %v", err)
		}
		return
	}

	request := r.Clone(r.Context())
	url := *r.URL
	url.Path = path
	request.URL = &url
	apiHandler.ServeHTTP(w, request)
}

func initialize() error {
	initializeLock.Lock()
	defer initializeLock.Unlock()
	if apiHandler != nil {
		return nil
	}

	db, err := backend.OpenDB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := backend.Initialize(ctx, db); err != nil {
		db.Close()
		return err
	}
	apiHandler = backend.NewHandler(db)
	return nil
}
