package main

import (
	"chirpy/internal/api"
	"chirpy/internal/database"
	"database/sql"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	_ = godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	platformType := os.Getenv("PLATFORM")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	cfg := api.Config{DbQueries: dbQueries, Platform: platformType}
	mux := http.NewServeMux()
	mux.Handle(
		"/app/",
		cfg.MiddlewareMetricsInc(
			http.StripPrefix(
				"/app",
				http.FileServer(http.Dir("./")),
			),
		),
	)
	mux.HandleFunc("GET /admin/metrics", cfg.MetricsEndpoint)
	mux.HandleFunc("POST /admin/reset", cfg.ResetDatabaseEndpoint)
	mux.HandleFunc("GET /admin/healthz", api.ReadyEndpoint)
	mux.HandleFunc("POST /api/chirps", cfg.ChirpsEndpoint)
	mux.HandleFunc("POST /api/users", cfg.CreateUserEndpoint)
	server := http.Server{Addr: ":8080", Handler: mux}
	if err := server.ListenAndServe(); err != nil {
		return
	}
}
