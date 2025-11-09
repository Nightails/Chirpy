package api

import (
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Metrics Handlers

func HandleOKRequest(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		return
	}
}

func (cfg *Config) DisplayMetrics(w http.ResponseWriter, req *http.Request) {
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

func (cfg *Config) ResetDatabase(w http.ResponseWriter, req *http.Request) {
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
	if err := cfg.DbQueries.RemoveAllChirps(req.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error resetting database")
		return
	}
	respondWithJSON(w, http.StatusOK, "Database reset")
}

// Chirps Handlers

func (cfg *Config) CreateChirp(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	// Request
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Authenticate
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}
	userID, err := auth.ValidateJWT(bearerToken, cfg.BearerToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User not authorized")
		return
	}

	// Create chirp
	if len(params.Body) <= 140 {
		filteredBody := filterProfanity(params.Body)
		chirp, err := cfg.DbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Body:      filteredBody,
			UserID:    userID,
		})
		if err != nil {
			errMessage := fmt.Sprintf("Error creating chirp: %v", err)
			respondWithError(w, http.StatusInternalServerError, errMessage)
			return
		}

		// Response
		type chirpResponse struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}
		resp := chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		data, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(data); err != nil {
			return
		}
	} else {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
	}
}

func (cfg *Config) GetChirps(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.DbQueries.GetChirps(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting chirps")
		return
	}

	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}
	resp := make([]chirpResponse, 0, len(chirps))
	for _, chirp := range chirps {
		resp = append(resp, chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func (cfg *Config) GetChirpByID(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "Missing chirp ID")
		return
	}

	chirp, err := cfg.DbQueries.GetChirpByID(req.Context(), uuid.MustParse(id))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error getting chirp")
		return
	}

	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}
	resp := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func (cfg *Config) DeleteChirpByID(w http.ResponseWriter, req *http.Request) {
	// Authorization
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}
	userID, err := auth.ValidateJWT(bearerToken, cfg.BearerToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User not authorized")
		return
	}

	// Check for chirp ID
	id := req.PathValue("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "Missing chirp ID")
		return
	}

	// Verify user ownership
	chirp, err := cfg.DbQueries.GetChirpByID(req.Context(), uuid.MustParse(id))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error getting chirp")
		return
	}
	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "User not authorized")
		return
	}

	// Delete chirp
	if err := cfg.DbQueries.RemoveChirpByID(req.Context(), uuid.MustParse(id)); err != nil {
		respondWithError(w, http.StatusNotFound, "Error deleting chirp")
		return
	}

	respondWithJSON(w, http.StatusNoContent, "Chirp deleted")
}

// Users Handlers

func (cfg *Config) RegisterUser(w http.ResponseWriter, req *http.Request) {
	// Request
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		errMessage := fmt.Sprintf("Error hashing password: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
	}

	// Create user
	user, err := cfg.DbQueries.CreateUser(req.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		errMessage := fmt.Sprintf("Error creating user: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	respondWithUserJSON(w, http.StatusCreated, user)
}

func (cfg *Config) LoginUser(w http.ResponseWriter, req *http.Request) {
	// Request
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Get user
	user, err := cfg.DbQueries.GetUserByEmail(req.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	// Check password
	if !auth.CheckPasswordHash(params.Password, user.HashedPassword) {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	// Generate JWT accessToken, expires in 1 hour
	accessToken, err := auth.MakeJWT(user.ID, cfg.BearerToken, 3600*time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate JWT")
		return
	}

	// Generate refreshToken, expires in 60 days
	refreshToken, err := auth.MakeRefreshToken()
	if _, err := cfg.DbQueries.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	// Response
	type userResponse struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		IsChirpyRed  bool      `json:"is_chirpy_red"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}
	resp := userResponse{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        accessToken,
		RefreshToken: refreshToken,
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func (cfg *Config) UpdateUser(w http.ResponseWriter, req *http.Request) {
	// Request Header
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.BearerToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User not authorized")
		return
	}

	// Request Body
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Update password
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		errMessage := fmt.Sprintf("Error hashing password: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
	}

	// Update user
	if err := cfg.DbQueries.UpdateUser(req.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          params.Email,
		HashedPassword: hashedPassword,
		UpdatedAt:      time.Now(),
	}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user")
		return
	}

	// Get user
	user, err := cfg.DbQueries.GetUserByID(req.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting user")
		return
	}

	respondWithUserJSON(w, http.StatusOK, user)
}

// Auth Handlers

func (cfg *Config) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	refreshToken, err := cfg.DbQueries.GetRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	if refreshToken.ExpiresAt.Before(time.Now()) {
		respondWithError(w, http.StatusUnauthorized, "Refresh token expired")
		return
	}
	if refreshToken.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "Refresh token revoked")
		return
	}

	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.BearerToken, 3600*time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate JWT")
		return
	}

	type userResponse struct {
		Token string `json:"token"`
	}
	resp := userResponse{
		Token: accessToken,
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func (cfg *Config) RevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
	}
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	if err := cfg.DbQueries.RevokeRefreshToken(r.Context(), database.RevokeRefreshTokenParams{
		Token: token,
		RevokedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
	}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke refresh token")
		return
	}

	respondWithJSON(w, http.StatusNoContent, "Refresh token revoked")
}

// Webhook Handlers

func (cfg *Config) ChirpyRedWebhook(w http.ResponseWriter, r *http.Request) {
	// Authorization
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if apiKey != cfg.APIKey {
		respondWithError(w, http.StatusUnauthorized, "Invalid API-Key")
		return
	}

	// Request
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		errMessage := fmt.Sprintf("Error decoding parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, errMessage)
		return
	}

	// Check for event
	if params.Event != "user.upgraded" {
		respondWithError(w, http.StatusNoContent, "Invalid event")
		return
	}

	// Get user ID
	userID := uuid.MustParse(params.Data.UserID)

	// Update users to be chirpy red
	if err := cfg.DbQueries.SetChirpyRedUserByID(r.Context(), database.SetChirpyRedUserByIDParams{
		ID:          userID,
		IsChirpyRed: true,
	}); err != nil {
		respondWithError(w, http.StatusNotFound, "Error updating user")
		return
	}
	respondWithJSON(w, http.StatusNoContent, "User upgraded")

}
