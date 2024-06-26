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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"runout/internal/config"
	"runout/internal/domain"
	"runout/internal/handlers"
	"runout/internal/models"
	"runout/internal/services"
	"runout/pkg/httpserver"
	"runout/pkg/logger"
)

// TODO: split to handlers/services/repositories
// TODO: add tests
// TODO: fix all slog messages
// TODO: удалять треды через сутки в фоне

//go:embed front/dist/*
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
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(&models.Need{})
	if err != nil {
		log.Error("AutoMigrate", logger.Error(err))
		return
	}

	OAIConfig := openai.DefaultConfig(cfg.OpenAI.APIKey)
	OAIConfig.AssistantVersion = "v2"
	oaiClient := openai.NewClientWithConfig(OAIConfig)

	voiceToTextService := services.NewVoiceToTextService(oaiClient)
	assistanRunnerService := services.NewAssistantManager(oaiClient, db, cfg.OpenAI.AssistantID)

	const audioChanSize = 1000
	audioChan := make(chan domain.Audio, audioChanSize)
	const reqChanSize = 1000
	reqChan := make(chan string, reqChanSize)

	go func() {
		for audio := range audioChan {
			log.Info("Audio received", slog.String("format", audio.Format))
			text, err := voiceToTextService.ProcessVoice(ctx, audio)
			if err != nil {
				fileName, fnErr := saveAudio(audio.Data, audio.Format)
				if fnErr != nil {
					log.Error("saveAudio", logger.Error(fnErr))
				}
				log.Error(fmt.Sprintf("ProcessVoice: %v", fileName), logger.Error(err))

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
	router.Path("/api/needs").Methods(http.MethodPost).Handler(handlers.AddNeed(audioChan))
	router.Path("/api/needs").Methods(http.MethodGet).Handler(handlers.ListNeeds(db))
	router.Path("/api/needs").Methods(http.MethodDelete).Handler(handlers.ClearNeeds(db))
	router.Path("/api/needs/{id}").Methods(http.MethodDelete).Handler(handlers.DeleteOne(db))

	static, err := fs.Sub(static, "front/dist")
	if err != nil {
		log.Error("Sub", logger.Error(err))
		os.Exit(1)
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

func saveAudio(data []byte, format string) (string, error) {
	audioID, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("NewV7: %w", err)
	}

	fineName := fmt.Sprintf("%v.%v", audioID, format)

	//nolint:gofumpt,gomnd,gosec
	err = os.WriteFile(
		fineName,
		data,
		0660,
	)
	if err != nil {
		return "", fmt.Errorf("WriteFile: %w", err)
	}

	return fineName, nil
}
