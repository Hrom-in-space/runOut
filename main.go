package main

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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

	"runout/internal/config"
	"runout/internal/db"
	"runout/internal/utils"
)

// TODO: add Assistant worker
// TODO: split to handlers/services/repositories
// TODO: add tests
// TODO: fix all slog messages
// TODO: удалять треды через сутки в фоне

//go:embed front/*
var static embed.FS

//nolint:funlen,cyclop
func main() {
	cfg, err := config.New()
	if err != nil {
		slog.Error("config.New", err)
		os.Exit(1)
	}

	ctx := context.Background()
	log.SetFlags(log.Ltime | log.Lshortfile)
	// TODO: add config

	const audioChanSize = 1000
	audioChan := make(chan Audio, audioChanSize)

	// Database
	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PG.Host, cfg.PG.Port, cfg.PG.Username, cfg.PG.Password, cfg.PG.Database)
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
	}

	// OpenAI client
	oaiClient := openai.NewClient(cfg.OpenAI.APIKey)

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
					AssistantID: cfg.OpenAI.AssistantID,
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

	router := mux.NewRouter()
	router.Path("/needs").Methods(http.MethodPost).Handler(addNeed(audioChan))
	router.Path("/needs").Methods(http.MethodGet).Handler(listNeeds(dbPool))

	static, err := fs.Sub(static, "front")
	if err != nil {
		slog.Error("Sub", err)
		os.Exit(1) //nolint:gocritic
	}
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServerFS(static)))
	// TODO: graceful shutdown
	slog.Info("Server is running on http://localhost:" + cfg.Port)
	httpServer := http.Server{
		ReadHeaderTimeout: 5 * time.Second, //nolint:gomnd
		Addr:              ":" + cfg.Port,
		Handler:           router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	err = httpServer.ListenAndServe()
	if err != nil {
		slog.Error("ListenAndServe: ", err)
	}
}

func addNeed(audioCh chan<- Audio) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		// enableCors(&w)
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

		contentType := req.Header.Get("Content-Type")
		format := strings.Replace(contentType, "audio/", "", 1)
		if !utils.InSlice(audioFormats, format) {
			slog.Error("wrong Content-Type", slog.String("content-type", contentType))
			writer.WriteHeader(http.StatusBadRequest)
			// TODO: return information about error
			return
		}

		data, err := io.ReadAll(req.Body)
		if err != nil {
			slog.Error("ReadAll", err)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}
		defer req.Body.Close()

		audioCh <- Audio{
			Data:   data,
			Format: format,
		}
		slog.Info("Audio added", slog.String("format", format))

		writer.WriteHeader(http.StatusOK)
	}
}

func listNeeds(pool *sql.DB) http.HandlerFunc {
	return func(respWriter http.ResponseWriter, req *http.Request) {
		q := db.New(pool)
		needs, err := q.ListNeeds(req.Context())
		if err != nil {
			slog.Error("ListNeeds", err)
			respWriter.WriteHeader(http.StatusInternalServerError)

			return
		}

		respWriter.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(respWriter).Encode(needs)
		if err != nil {
			slog.Error("Encode needs", err)
			respWriter.WriteHeader(http.StatusInternalServerError)
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

//nolint:cyclop
func (r *RunManager) Run(ctx context.Context) error {
	const defaultTimeout = time.Millisecond * 200
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

			_, _ = r.Client.SubmitToolOutputs(ctx, r.TheradID, r.RunID, openai.SubmitToolOutputsRequest{
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
		time.Sleep(defaultTimeout)
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
