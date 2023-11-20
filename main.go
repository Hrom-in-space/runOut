package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	// TODO: add config

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"localhost", 5432, "postgres", "postgres", "mydatabase")
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		slog.Error("db open", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("db close", err)
		}
	}()

	err = db.Ping()
	if err != nil {
		slog.Error("no db connection: ", err)
		os.Exit(1)
	}

	// TODO: handler GET /needs
	// TODO: handler POST /needs
	router := mux.NewRouter()
	router.HandleFunc("/", addNeeds)

	// TODO: graceful shutdown
	// TODO: get port from config
	slog.Info("Server is running on http://localhost:8080")
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		slog.Error("ListenAndServe: ", err)
	}
}

func addNeeds(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
}
