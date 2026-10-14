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
