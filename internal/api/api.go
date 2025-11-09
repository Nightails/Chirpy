package api

import (
	"chirpy/internal/database"
	"net/http"
	"sync/atomic"
)

type Config struct {
	DbQueries      *database.Queries
	FileserverHits atomic.Int32
	Platform       string
	BearerToken    string
	APIKey         string
}

func (cfg *Config) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}
