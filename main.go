package main

import (
	"bytes"
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
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"

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
	oaiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	go func() {
		for audio := range audioChan {
			slog.Info("Audio received", slog.String("format", audio.Format))
			request := openai.AudioRequest{
				Model:    "whisper-1",
				Reader:   bytes.NewReader(audio.Data),
				FilePath: "need." + audio.Format,
			}
			resp, err := oaiClient.CreateTranscription(ctx, request)
			if err != nil {
				slog.Error("CreateTranscription", err)
				continue
			}
			slog.Info("Transcription created", slog.String("text", resp.Text))

			createThreadAndRunRequest := openai.CreateThreadAndRunRequest{
				RunRequest: openai.RunRequest{
					AssistantID: os.Getenv("OPENAI_ASSISTANT_ID"),
				},
				Thread: openai.ThreadRequest{
					Messages: []openai.ThreadMessage{
						{
							Role:    openai.ThreadMessageRoleUser,
							Content: resp.Text,
						},
					},
				},
			}
			runResponse, err := oaiClient.CreateThreadAndRun(ctx, createThreadAndRunRequest)
			if err != nil {
				slog.Error("CreateThreadAndRun", err)
				continue
			}
			slog.Info("Run created", slog.String("run_id", runResponse.ID))
			// slog.Info("Run created", slog.String("response", fmt.Sprintf("%#v", runResponse)))

			runMngr := RunManager{
				TheradID: runResponse.ThreadID,
				RunID:    runResponse.ID,
				Client:   oaiClient,
				Pool:     dbPool,
			}
			err = runMngr.Run(ctx)
			if err != nil {
				slog.Error("RunManager.Run", err)
				continue
			}
			slog.Info("Run completed", slog.String("run_id", runResponse.ID))
		}
	}()

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
			slog.Error("wrong Content-Type", slog.String("content-type", contentType))
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
		slog.Info("Audio added", slog.String("format", format))

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

type RunManager struct {
	TheradID string
	RunID    string
	Client   *openai.Client
	Pool     *sql.DB
}

func (r *RunManager) Run(ctx context.Context) error {
	for {
		run, err := r.Client.RetrieveRun(ctx, r.TheradID, r.RunID)
		if err != nil {
			return fmt.Errorf("retrieve run: %w", err)
		}
		slog.Info("Run status", slog.String("status", string(run.Status)))

		switch run.Status {
		case openai.RunStatusQueued, openai.RunStatusInProgress:
			continue
		case openai.RunStatusRequiresAction:
			// TODO: partial success
			var successIDs []string
			for _, call := range run.RequiredAction.SubmitToolOutputs.ToolCalls {
				if call.Function.Name == "addNeed" {
					need, err := parseNeedsArgs(call.Function.Arguments)
					slog.Info(
						"raw args",
						slog.String("example", fmt.Sprintf("|%v|", "abc")),
						slog.String("raw args", fmt.Sprintf("|%v|", call.Function.Arguments)))
					if err != nil {
						return fmt.Errorf("parse needs args: %w", err)
					}
					err = AddNeed(ctx, r.Pool, need)
					if err != nil {
						return fmt.Errorf("add need: %w", err)
					}
					successIDs = append(successIDs, call.ID)
				}
			}

			toolOutputs := make([]openai.ToolOutput, len(successIDs))
			for i, id := range successIDs {
				toolOutputs[i] = openai.ToolOutput{
					ToolCallID: id,
					Output:     "success",
				}
			}

			run, err = r.Client.SubmitToolOutputs(ctx, r.TheradID, r.RunID, openai.SubmitToolOutputsRequest{
				ToolOutputs: toolOutputs,
			})
		case openai.RunStatusCompleted:
			return nil
		case openai.RunStatusFailed:
			return fmt.Errorf("run failed at %v with %v:%v", run.FailedAt, run.LastError.Code, run.LastError.Message)
		case openai.RunStatusExpired:
			return fmt.Errorf("run expired")
		case openai.RunStatusCancelling:
			return fmt.Errorf("run cancelling")
		}
		time.Sleep(time.Millisecond * 200)
	}
}

func AddNeed(ctx context.Context, pool *sql.DB, need string) error {
	q := db.New(pool)
	err := q.CreateNeed(ctx, need)
	if err != nil {
		return fmt.Errorf("create need: %w", err)
	}
	return nil
}

// parseNeedsArgs parses arguments for addNeed function.
func parseNeedsArgs(arg string) (string, error) {
	var need Need

	err := json.Unmarshal([]byte(arg), &need)
	if err != nil {
		return "", fmt.Errorf("unmarshal needs: %w", err)
	}
	return need.Name, nil
}

type Need struct {
	Name string `json:"name"`
}
