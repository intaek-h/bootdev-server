package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type payload struct {
		Body string `json:"body"`
	}
	type responseBody struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	body := payload{}
	err := decoder.Decode(&body)

	if err != nil {
		log.Printf("JSON 페이로드를 디코딩하는 중에 오류가 발생했습니다: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const maxChirpLength = 140
	if len(body.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp 너무 깁니다.")
		return
	}

	bannedWords := []string{"kerfuffle", "****", "Kerfuffle", "****", "sharbert", "****", "Sharbert", "****", "fornax", "****", "Fornax", "****"}
	replacer := strings.NewReplacer(bannedWords...)
	cleanedBody := replacer.Replace(body.Body)

	respondWithJSON(w, http.StatusOK, responseBody{CleanedBody: cleanedBody})
}

// 문자열 변환 다른 로직 예시
func cleanupChirp(body string, badWords map[string]struct{}) string {
	words := strings.Split(body, " ")

	for i, word := range words {
		lowercase := strings.ToLower(word)

		if _, ok := badWords[lowercase]; ok {
			words[i] = "****"
		}
	}

	cleaned := strings.Join(words, " ")

	return cleaned
}
