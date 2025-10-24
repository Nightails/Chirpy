package api

import (
	"chirpy/internal/database"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func ReadyEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		return
	}
}

func (cfg *Config) MetricsEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	metrics := fmt.Sprintf(
		"<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>",
		cfg.FileserverHits.Load(),
	)
	if _, err := w.Write([]byte(metrics)); err != nil {
		return
	}
}

func (cfg *Config) ResetDatabaseEndpoint(w http.ResponseWriter, req *http.Request) {
	// Only allow reset on a development platform
	if cfg.Platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Not authorized")
		return
	}

	// Reset database
	if err := cfg.DbQueries.RemoveAllUsers(req.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error resetting database")
		return
	}
	respondWithJSON(w, http.StatusOK, "Database reset")
}

func ValidateChirpEndpoint(w http.ResponseWriter, req *http.Request) {
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

func (cfg *Config) CreateUserEndpoint(w http.ResponseWriter, req *http.Request) {
	// Request
	type parameters struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Create user
	user, err := cfg.DbQueries.CreateUser(req.Context(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     params.Email,
	})
	if err != nil {
		errMessage := fmt.Sprintf("Error creating user: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Response
	type userResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	resp := userResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(data); err != nil {
		return
	}
}
