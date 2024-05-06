package services

import (
	"bytes"
	"context"

	"github.com/sashabaranov/go-openai"

	"runout/internal/domain"
)

type VoiceToTextService struct {
	oaiClient *openai.Client
}

func NewVoiceToTextService(oaiClient *openai.Client) *VoiceToTextService {
	return &VoiceToTextService{
		oaiClient: oaiClient,
	}
}

func (v *VoiceToTextService) ProcessVoice(ctx context.Context, voice domain.Audio) (string, error) {
	request := openai.AudioRequest{
		Model:    "whisper-1",
		Reader:   bytes.NewReader(voice.Data),
		FilePath: "need." + voice.Format,
		Language: "ru",
	}
	resp, err := v.oaiClient.CreateTranscription(ctx, request)
	if err != nil {
		return "", err
	}

	return resp.Text, nil
}
