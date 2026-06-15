package ai

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// MockLLM implements the llms.Model interface to allow testing
// the pipeline without making live API calls to an LLM provider.
type MockLLM struct {
	Response string
}

// GenerateContent simulates generating content by returning the hardcoded Response.
func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: m.Response,
			},
		},
	}, nil
}

// Call simulates a simple text completion call by returning the hardcoded Response.
func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return m.Response, nil
}
