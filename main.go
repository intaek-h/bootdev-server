package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8080"
	var mux = http.ServeMux{}
	var corsMux = middlewareCors(&mux)
	var server = &http.Server{Handler: corsMux, Addr: ":" + port}

	log.Printf("%s 포트에서 서버를 시작합니다.\n", port)

	var err = server.ListenAndServe()

	log.Fatal(err)
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
