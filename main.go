package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	// TODO: add config

	// TODO: database
	// TODO: handler GET /needs
	// TODO: handler POST /needs
	router := mux.NewRouter()
	router.HandleFunc("/", addNeeds)

	// TODO: graceful shutdown
	// TODO: get port from config
	slog.Info("Server is running on http://localhost:8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		slog.Error("ListenAndServe: ", err)
	}
}

func addNeeds(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
}
