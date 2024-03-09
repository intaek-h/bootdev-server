package main

import "net/http"

func (cfg *apiConfig) handlerMetricsReset(w http.ResponseWriter, _ *http.Request) {
	cfg.fileserverHits = 0
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(http.StatusText(http.StatusOK))) // OK
}
