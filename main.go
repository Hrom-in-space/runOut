package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"

	"runout/internal/config"
	"runout/internal/domain"
	"runout/internal/handlers"
	"runout/internal/repo"
	"runout/internal/services"
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

	oaiClient := openai.NewClient(cfg.OpenAI.APIKey)
	voiceToTextService := services.NewVoiceToTextService(oaiClient)
	assistanRunnerService := services.NewAssistantManager(oaiClient, trManager, repo, cfg.OpenAI.AssistantID)

	const audioChanSize = 1000
	audioChan := make(chan domain.Audio, audioChanSize)
	const reqChanSize = 1000
	reqChan := make(chan string, reqChanSize)

	go func() {
		for audio := range audioChan {
			log.Info("Audio received", slog.String("format", audio.Format))
			text, err := voiceToTextService.ProcessVoice(ctx, audio)
			if err != nil {
				log.Error("ProcessVoice", logger.Error(err))
				continue
			}
			log.Info("Transcription created", slog.String("text", text))
			reqChan <- text
		}
	}()

	go func() {
		for text := range reqChan {
			err := assistanRunnerService.Run(ctx, text)
			if err != nil {
				log.Error("Run", logger.Error(err))
			}
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
