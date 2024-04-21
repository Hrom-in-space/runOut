package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sashabaranov/go-openai"

	"runout/pkg/logger"
	"runout/pkg/pg"
)

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
