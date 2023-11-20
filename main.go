package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"runout/internal/db"
	"runout/internal/utils"
)

// TODO: add Assistant worker
// TODO: split to handlers/services/repositories
// TODO: add tests
// TODO: fix all slog messages
// TODO: удалять треды через сутки в фоне

func main() {
	ctx := context.Background()
	log.SetFlags(log.Ltime | log.Lshortfile)
	// TODO: add config

	audioChan := make(chan Audio, 1000)

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
	// oaiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	// TODO: handler GET /needs
	// TODO: handler POST /needs
	router := mux.NewRouter()
	router.Path("/needs").Methods(http.MethodPost).Handler(addNeed(audioChan))
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

func addNeed(ch chan<- Audio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: security check real file type https://github.com/h2non/filetype
		audioFormats := []string{
			"flac",
			"mp3",
			"mp4",
			"mpeg",
			"mpga",
			"m4a",
			"ogg",
			"wav",
			"webm",
		}

		contentType := r.Header.Get("Content-Type")
		format := strings.Replace(contentType, "audio/", "", 1)
		if !utils.InSlice(audioFormats, format) {
			slog.Error("wrong Content-Type", contentType)
			w.WriteHeader(http.StatusBadRequest)
			// TODO: return information about error
			return
		}

		data, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("ReadAll", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		ch <- Audio{
			Data:   data,
			Format: format,
		}

		w.WriteHeader(http.StatusOK)
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

type Audio struct {
	Format string
	Data   []byte
}

// func fn(oaiClient *openai.Client, pool *sql.DB) {
// 	request := openai.AudioRequest{
// 		Model:    "whisper-1",
// 		FilePath: "need.mp3",
// 		Reader:   r.Body,
// 	}
// 	resp, err := oaiClient.CreateTranscription(r.Context(), request)
// 	if err != nil {
// 		slog.Error("ListModels", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
//
// 	// TODO: add request to Assistant
//
// 	w.Header().Set("Content-Type", "application/json")
// 	err = json.NewEncoder(w).Encode(resp.Text)
// 	if err != nil {
// 		slog.Error("Encode models", err)
// 	}
//
// 	// q := db.New(pool)
// 	// err = q.CreateNeed(r.Context(), resp.Text)
// 	// if err != nil {
// 	// 	slog.Error("CreateNeed", err)
// 	// 	w.WriteHeader(http.StatusInternalServerError)
// 	// 	return
// 	// }
// }
