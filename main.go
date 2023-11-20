package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"

	"runout/internal/db"
)

func main() {
	ctx := context.Background()
	log.SetFlags(log.Ltime | log.Lshortfile)
	// TODO: add config

	// Database
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

	// OpenAI client
	oaiClient := openai.NewClient("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")

	// TODO: handler GET /needs
	// TODO: handler POST /needs
	router := mux.NewRouter()
	router.Path("/needs").Methods(http.MethodPost).Handler(addNeed(dbPool, oaiClient))
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

func addNeed(pool *sql.DB, oaiClient *openai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: add all openai supported formats
		if r.Header.Get("Content-Type") != "audio/mp3" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		request := openai.AudioRequest{
			Model:    "whisper-1",
			FilePath: "need.mp3",
			Reader:   r.Body,
		}
		resp, err := oaiClient.CreateTranscription(r.Context(), request)
		if err != nil {
			slog.Error("ListModels", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// TODO: add request to Assistant

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp.Text)
		if err != nil {
			slog.Error("Encode models", err)
		}

		// q := db.New(pool)
		// err = q.CreateNeed(r.Context(), resp.Text)
		// if err != nil {
		// 	slog.Error("CreateNeed", err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }
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

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(needs)
		if err != nil {
			slog.Error("Encode needs", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
