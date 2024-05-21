package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sashabaranov/go-openai"

	"runout/internal/repo"
	"runout/pkg/logger"
	"runout/pkg/pg"
)

type AssistantManager struct {
	oaiClient   *openai.Client
	trm         *pg.TxManager
	repo        *repo.Repo
	assistantID string
}

func NewAssistantManager(
	oaiClient *openai.Client,
	trm *pg.TxManager,
	repo *repo.Repo,
	assistantID string,
) *AssistantManager {
	return &AssistantManager{
		oaiClient:   oaiClient,
		trm:         trm,
		repo:        repo,
		assistantID: assistantID,
	}
}

//nolint:cyclop,funlen
func (m *AssistantManager) Run(ctx context.Context, text string) error {
	log := logger.FromCtx(ctx)

	createThreadAndRunRequest := openai.CreateThreadAndRunRequest{
		RunRequest: openai.RunRequest{
			AssistantID: m.assistantID,
		},
		Thread: openai.ThreadRequest{
			Messages: []openai.ThreadMessage{
				{
					Role:    openai.ThreadMessageRoleUser,
					Content: text,
				},
			},
		},
	}
	runResponse, err := m.oaiClient.CreateThreadAndRun(ctx, createThreadAndRunRequest)
	if err != nil {
		return fmt.Errorf("create thread and run: %w", err)
	}

	const defaultTimeout = time.Millisecond * 300
	for {
		run, err := m.oaiClient.RetrieveRun(ctx, runResponse.ThreadID, runResponse.ID)
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
					log.Info("func addNeed", slog.String("raw args", call.Function.Arguments))
					need, err := parseNeedsArgs(call.Function.Arguments)
					if err != nil {
						return fmt.Errorf("parse needs args: %w", err)
					}
					err = AddNeed(ctx, m.trm, m.repo, need)
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

			_, _ = m.oaiClient.SubmitToolOutputs(ctx, runResponse.ThreadID, runResponse.ID, openai.SubmitToolOutputsRequest{
				ToolOutputs: toolOutputs,
			})
		case openai.RunStatusCompleted:
			return nil
		case openai.RunStatusFailed:
			return fmt.Errorf("run failed at %v with %v:%v", run.FailedAt, run.LastError.Code, run.LastError.Message)
		case openai.RunStatusExpired:
			return fmt.Errorf("run expired")
		case openai.RunStatusCancelling, openai.RunStatusCancelled:
			return fmt.Errorf("run cancelling")
		default:
			return fmt.Errorf("unknown run status: %v", run.Status)
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
