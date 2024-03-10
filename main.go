package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/intaek-h/bootdev-server/internal/database"
)

type apiConfig struct {
	fileserverHits int
	DB             *database.DB
}

func main() {
	debugMode := flag.Bool("debug", false, "디버그 모드를 활성화합니다.")
	flag.Parse()

	if !*debugMode {
		log.Println("운영 모드는 아직 지원되지 않습니다.")
		return
	}

	db, err := database.NewDB("database.json")
	if err != nil {
		log.Fatalf("데이터베이스를 열 수 없습니다: %s\n", err)
		return
	}

	const filePathRoot = "."
	const port = "8080"

	var mux = http.NewServeMux()
	var corsMux = middlewareCors(mux)
	var server = &http.Server{Handler: corsMux, Addr: ":" + port}
	var cfg = &apiConfig{fileserverHits: 0, DB: db}

	mux.Handle("/app/*", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /api/metrics", cfg.handlerMetrics)
	mux.HandleFunc("GET /api/reset", cfg.handlerMetricsReset)
	mux.HandleFunc("POST /api/chirps", cfg.handlerPostChirp)
	mux.HandleFunc("GET /api/chirps", cfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpId}", cfg.handlerGetChirp)
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("POST /api/login", cfg.handlerLogin)

	log.Printf("%s 포트에서 서버를 시작합니다.\n", port)

	err = server.ListenAndServe()

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
