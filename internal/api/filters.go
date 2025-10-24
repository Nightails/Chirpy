package api

import (
	"encoding/json"
	"log"
	"net/http"
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
