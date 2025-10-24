package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	dbQueries      *database.Queries
	fileserverHits atomic.Int32
}

func main() {
	_ = godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	apiCfg := apiConfig{dbQueries: dbQueries}
	mux := http.NewServeMux()
	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(
			http.StripPrefix(
				"/app",
				http.FileServer(http.Dir("./")),
			),
		),
	)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsEndpoint)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetricsEndpoint)
	mux.HandleFunc("GET /admin/healthz", readyEndpoint)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpEndpoint)
	server := http.Server{Addr: ":8080", Handler: mux}
	if err := server.ListenAndServe(); err != nil {
		return
	}
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func readyEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		return
	}
}

func (cfg *apiConfig) metricsEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	metrics := fmt.Sprintf(
		"<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>",
		cfg.fileserverHits.Load(),
	)
	if _, err := w.Write([]byte(metrics)); err != nil {
		return
	}
}

func (cfg *apiConfig) resetMetricsEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
	metrics := fmt.Sprintf("Hits reseted to %d", cfg.fileserverHits.Load())
	if _, err := w.Write([]byte(metrics)); err != nil {
		return
	}
}

func validateChirpEndpoint(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	if len(params.Body) <= 140 {
		filteredBody := filterProfanity(params.Body)
		respondWithJSON(w, http.StatusOK, filteredBody)
		return
	} else {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	log.Println(message)
	resp := errorResponse{Error: message}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	type jsonResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	resp := jsonResponse{CleanedBody: payload.(string)}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func filterProfanity(body string) string {
	banWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Fields(body)
	var filteredWords []string
	for _, w := range words {
		isBanned := false
		lowerW := strings.ToLower(w)
		for _, banWord := range banWords {
			if lowerW == banWord {
				isBanned = true
				break
			}
		}
		if isBanned {
			filteredWords = append(filteredWords, "****")
		} else {
			filteredWords = append(filteredWords, w)
		}
	}
	return strings.Join(filteredWords, " ")
}
