package api

import (
	"chirpy/internal/database"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

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

func respondWithUserJSON(w http.ResponseWriter, code int, user database.User) {
	type userResponse struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}
	resp := userResponse{
		ID:          user.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		return
	}
}
