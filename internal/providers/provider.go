package providers

import (
	"context"
	"cookie-router/internal/models"
)

type Provider interface {
	ID() string
	Name() string
	DefaultModels() []string
	Test(ctx context.Context, cookie string) error
	GetAvailableModels(ctx context.Context, cookie string) ([]string, error)
	Chat(ctx context.Context, req *models.ChatCompletionRequest, cookie string) (*models.ChatCompletionResponse, error)
	ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error
}

type ImageDownloader interface {
	DownloadImage(ctx context.Context, cookie string, fileID string) ([]byte, error)
}

type VideoGenerator interface {
	GenerateVideo(ctx context.Context, cookie string, prompt, aspect string, duration int) ([]byte, string, error)
}

type AccountInfo struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
	Plan  string `json:"plan,omitempty"`
}

type AccountInfoDetector interface {
	DetectAccountInfo(ctx context.Context, cookie string) (*AccountInfo, error)
}

var Registry = make(map[string]Provider)

func Register(p Provider) {
	Registry[p.ID()] = p
}

func Get(id string) Provider {
	return Registry[id]
}

func All() map[string]Provider {
	return Registry
}
