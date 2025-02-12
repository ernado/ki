package main

import "net/http"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
