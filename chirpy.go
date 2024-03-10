package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/intaek-h/bootdev-server/internal/database"
)

func (cfg *apiConfig) handlerPostChirp(w http.ResponseWriter, r *http.Request) {
	type payload struct {
		Body string `json:"body"`
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

	chirp, err := cfg.DB.CreateChirp(cleanedBody)
	if err != nil {
		log.Printf("Chirp를 데이터베이스에 저장하는 중에 오류가 발생했습니다: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusCreated, chirp)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, _ *http.Request) {
	dbChirps, err := cfg.DB.GetChirps()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Chirps를 가져오는 중에 오류가 발생했습니다.")
		return
	}

	chirps := []database.Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, database.Chirp{Id: dbChirp.Id, Body: dbChirp.Body})
	}

	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].Id < chirps[j].Id
	})

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpId")
	dbChirp, err := cfg.DB.GetChirp(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp를 가져오는 중에 오류가 발생했습니다.")
		return
	}

	respondWithJSON(w, http.StatusOK, database.Chirp{Id: dbChirp.Id, Body: dbChirp.Body})
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	type responseBody struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}

	var user database.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err = cfg.DB.CreateUser(user.Email, user.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	respondWithJSON(w, http.StatusCreated, responseBody{Id: user.Id, Email: user.Email})
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	type requestBody struct {
		Email    string
		Password string
	}

	type responseBody struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}

	var body requestBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err := cfg.DB.GetUser(body.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	err = user.ComparePassword(body.Password)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	respondWithJSON(w, http.StatusOK, responseBody{Email: user.Email, Id: user.Id})
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
