package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	const filePathRoot = "."
	const port = "8080"

	var mux = http.NewServeMux()
	var corsMux = middlewareCors(mux)
	var server = &http.Server{Handler: corsMux, Addr: ":" + port}
	var cfg = &apiConfig{fileserverHits: 0}

	mux.Handle("/app/*", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /api/metrics", cfg.handlerMetrics)
	mux.HandleFunc("GET /api/reset", cfg.handlerMetricsReset)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	log.Printf("%s 포트에서 서버를 시작합니다.\n", port)

	var err = server.ListenAndServe()

	log.Fatal(err)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits)))
}

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
