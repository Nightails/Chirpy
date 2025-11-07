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
	bearerTokenSecret := os.Getenv("BEARER_TOKEN_SECRET")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	cfg := api.Config{
		DbQueries:   dbQueries,
		Platform:    platformType,
		BearerToken: bearerTokenSecret,
	}
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
	mux.HandleFunc("GET /admin/metrics", cfg.DisplayMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.ResetDatabase)
	mux.HandleFunc("GET /admin/healthz", api.HandleOKRequest)
	mux.HandleFunc("POST /api/chirps", cfg.CreateChirp)
	mux.HandleFunc("GET /api/chirps", cfg.GetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", cfg.GetChirpByID)
	mux.HandleFunc("DELETE /api/chirps/{id}", cfg.DeleteChirpByID)
	mux.HandleFunc("POST /api/users", cfg.RegisterUser)
	mux.HandleFunc("PUT /api/users", cfg.UpdateUser)
	mux.HandleFunc("POST /api/login", cfg.LoginUser)
	mux.HandleFunc("POST /api/refresh", cfg.RefreshTokenHandler)
	mux.HandleFunc("POST /api/revoke", cfg.RevokeRefreshToken)
	server := http.Server{Addr: ":8080", Handler: mux}
	if err := server.ListenAndServe(); err != nil {
		return
	}
}
