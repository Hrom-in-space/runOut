package services

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/sashabaranov/go-openai"

	"runout/internal/domain"
	"runout/internal/repo"
	"runout/pkg/logger"
	"runout/pkg/pg"
)

func AudioProcessor(
	ctx context.Context,
	audioChan <-chan domain.Audio,
	oaiClient *openai.Client,
	trm *pg.TxManager,
	repo *repo.Repo,
	assistantID string,
) {
	log := logger.FromCtx(ctx)

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
				AssistantID: assistantID,
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
			Trm:      trm,
			Repo:     repo,
		}
		err = runMngr.Run(ctx)
		if err != nil {
			log.Error("RunManager.Run", logger.Error(err))
			continue
		}
		log.Info("Run completed", slog.String("run_id", runResponse.ID))
	}
}
