package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/intaek-h/bootdev-server/internal/auth"
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
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	type responseBody struct {
		Id           int    `json:"id"`
		Email        string `json:"email"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
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

	defaultAccessExpiration := int(time.Duration(time.Hour * 24).Seconds()) // 24 hours in seconds (int)

	// if the user specified the expiration time, AND it's less than the default expiration time
	// then we'll use the user's expiration time
	if body.ExpiresInSeconds > 0 && body.ExpiresInSeconds < defaultAccessExpiration {
		defaultAccessExpiration = body.ExpiresInSeconds
	}

	token, err := auth.MakeJWT("chirpy_access", user.Id, cfg.jwtSecret, time.Duration(defaultAccessExpiration)*time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create JWT")
		return
	}

	defaultRefreshExpiration := time.Duration(time.Hour * 24 * 60) // 7 days in seconds (int)
	refreshToken, err := auth.MakeJWT("chirpy_refresh", user.Id, cfg.jwtSecret, defaultRefreshExpiration)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create JWT")
		return
	}

	respondWithJSON(w, http.StatusOK, responseBody{Email: user.Email, Id: user.Id, Token: token, RefreshToken: refreshToken})
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		Email string `json:"email"`
		Id    int    `json:"id"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token")
		return
	}

	subject, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	params := parameters{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}

	userIdInt, err := strconv.Atoi(subject)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could no parse user ID")
		return
	}

	user, err := cfg.DB.UpdateUser(userIdInt, params.Email, hashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user")
		return
	}

	respondWithJSON(w, http.StatusOK, response{Email: user.Email, Id: user.Id})
}

func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token")
		return
	}

	tokenData, err := jwt.ParseWithClaims(
		token,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.jwtSecret), nil
		},
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	issuer, err := tokenData.Claims.GetIssuer()
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error getting issuer")
		return
	}

	if issuer != "chirpy_refresh" {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	userId, err := tokenData.Claims.GetSubject()
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error getting user ID")
		return
	}

	jwtTime, err := tokenData.Claims.GetExpirationTime()
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error getting expiration time")
		return
	}
	if time.Until(jwtTime.Time) < 0 {
		respondWithError(w, http.StatusUnauthorized, "Token expired")
		return
	}

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
