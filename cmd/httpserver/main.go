package main

import (
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	})
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           http.DefaultServeMux,
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
