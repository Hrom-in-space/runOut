package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"

	"runout/internal/config"
	"runout/internal/domain"
	"runout/internal/handlers"
	"runout/internal/repo"
	"runout/pkg/httpserver"
	"runout/pkg/logger"
	"runout/pkg/pg"
)

// TODO: split to handlers/services/repositories
// TODO: add tests
// TODO: fix all slog messages
// TODO: удалять треды через сутки в фоне

//go:embed front/*
var static embed.FS

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

//nolint:funlen,cyclop
func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	log, err := logger.New("INFO", cfg.Logger.Format)
	if err != nil {
		panic(err)
	}
	ctx = logger.ToCtx(ctx, log)

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.PG.Username, cfg.PG.Password, net.JoinHostPort(cfg.PG.Host, cfg.PG.Port), cfg.PG.Database,
	)
	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Error("Unable to create connection pool", logger.Error(err))
		os.Exit(1)
	}
	err = dbPool.Ping(ctx)
	if err != nil {
		log.Error("Error database connection", logger.Error(err))
	}
	defer dbPool.Close()

	trManager := pg.NewTxManager(dbPool)
	repo := repo.New()

	// OpenAI client
	oaiClient := openai.NewClient(cfg.OpenAI.APIKey)

	const audioChanSize = 1000
	audioChan := make(chan domain.Audio, audioChanSize)
	go func() {
		for audio := range audioChan {
			log.Info("Audio received", slog.String("format", audio.Format))
			request := openai.AudioRequest{
				Model:    "whisper-1",
				Reader:   bytes.NewReader(audio.Data),
				FilePath: "need." + audio.Format,
			}
			resp, err := oaiClient.CreateTranscription(ctx, request)
			if err != nil {
				log.Error("CreateTranscription", logger.Error(err))
				continue
			}
			log.Info("Transcription created", slog.String("text", resp.Text))

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
				log.Error("CreateThreadAndRun", logger.Error(err))
				continue
			}

			runMngr := RunManager{
				ThreadID: runResponse.ThreadID,
				RunID:    runResponse.ID,
				Client:   oaiClient,
				Trm:      trManager,
				Repo:     repo,
			}
			err = runMngr.Run(ctx)
			if err != nil {
				log.Error("RunManager.Run", logger.Error(err))
				continue
			}
			log.Info("Run completed", slog.String("run_id", runResponse.ID))
		}
	}()

	router := mux.NewRouter()
	router.Path("/needs").Methods(http.MethodPost).Handler(handlers.AddNeed(audioChan))
	router.Path("/needs").Methods(http.MethodGet).Handler(handlers.ListNeeds(trManager, repo))

	static, err := fs.Sub(static, "front")
	if err != nil {
		log.Error("Sub", logger.Error(err))
		os.Exit(1) //nolint:gocritic
	}
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServerFS(static)))
	// TODO: graceful shutdown
	log.Info("Server is running on http://localhost:" + cfg.Port)

	httpServer := httpserver.New(ctx, router, cfg.Port)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Error("ListenAndServe", logger.Error(err))
	}
}

type RunManager struct {
	ThreadID string
	RunID    string
	Client   *openai.Client
	Trm      pg.Manager
	Repo     NeedAdder
}

//nolint:cyclop
func (r *RunManager) Run(ctx context.Context) error {
	log := logger.FromCtx(ctx)
	const defaultTimeout = time.Millisecond * 200
	for {
		run, err := r.Client.RetrieveRun(ctx, r.ThreadID, r.RunID)
		if err != nil {
			return fmt.Errorf("retrieve run: %w", err)
		}
		log.Info("Run status", slog.String("status", string(run.Status)))

		switch run.Status {
		case openai.RunStatusQueued, openai.RunStatusInProgress:
			continue
		case openai.RunStatusRequiresAction:
			// TODO: partial success
			var successIDs []string
			for _, call := range run.RequiredAction.SubmitToolOutputs.ToolCalls {
				if call.Function.Name == "addNeed" {
					need, err := parseNeedsArgs(call.Function.Arguments)
					log.Info(
						"raw args",
						slog.String("example", fmt.Sprintf("|%v|", "abc")),
						slog.String("raw args", fmt.Sprintf("|%v|", call.Function.Arguments)))
					if err != nil {
						return fmt.Errorf("parse needs args: %w", err)
					}
					err = AddNeed(ctx, r.Trm, r.Repo, need)
					if err != nil {
						return fmt.Errorf("add need in DB: %w", err)
					}
					log.Info("Need added", slog.String("name", need))
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

			_, _ = r.Client.SubmitToolOutputs(ctx, r.ThreadID, r.RunID, openai.SubmitToolOutputsRequest{
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

type NeedAdder interface {
	AddNeed(ctx context.Context, need string) error
}

func AddNeed(ctx context.Context, trm pg.Manager, repo NeedAdder, need string) error {
	if err := trm.Do(ctx, func(ctx context.Context) error {
		err := repo.AddNeed(ctx, need)
		if err != nil {
			return fmt.Errorf("error add need: %w", err)
		}

		return nil
	}); err != nil {
		return err
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
