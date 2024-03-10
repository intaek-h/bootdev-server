package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, message string) {
	if code > 499 {
		log.Printf("서버 오류: %s\n", message)
	}

	type errResp struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, errResp{Error: message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")

	data, err := json.Marshal(payload)

	if err != nil {
		log.Printf("JSON을 마샬하는 중에 오류가 발생했습니다: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)
	w.Write(data)
}
