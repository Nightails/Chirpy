package api

import "strings"

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
