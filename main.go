package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"runout/internal/db"
)

func main() {
	ctx := context.Background()
	log.SetFlags(log.Ltime | log.Lshortfile)
	// TODO: add config

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"localhost", 5432, "postgres", "postgres", "mydatabase")
	dbPool, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		slog.Error("dbPool open", err)
	}
	defer func() {
		if err := dbPool.Close(); err != nil {
			slog.Error("dbPool close", err)
		}
	}()
	err = dbPool.Ping()
	if err != nil {
		slog.Error("no dbPool connection: ", err)
		os.Exit(1)
	}

	// TODO: handler GET /needs
	// TODO: handler POST /needs
	router := mux.NewRouter()
	// router.HandleFunc("/needs", addNeeds(dbPool))
	router.Path("/needs").Methods(http.MethodPost).Handler(addNeed(dbPool))
	router.Path("/needs").Methods(http.MethodGet).Handler(listNeeds(dbPool))

	// TODO: graceful shutdown
	// TODO: get port from config
	slog.Info("Server is running on http://localhost:8080")
	httpServer := http.Server{
		Addr:    ":8080",
		Handler: router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	err = httpServer.ListenAndServe()
	if err != nil {
		slog.Error("ListenAndServe: ", err)
	}
}

func addNeed(pool *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		needName := r.URL.Query().Get("n")
		if needName == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		q := db.New(pool)
		err := q.CreateNeed(r.Context(), needName)
		if err != nil {
			slog.Error("CreateNeed", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "need %v added", needName)
	}
}

func listNeeds(pool *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := db.New(pool)
		needs, err := q.ListNeeds(r.Context())
		if err != nil {
			slog.Error("ListNeeds", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "need required: %v", needs)
	}
}
