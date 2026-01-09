package tendo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zoobzio/capitan"
	"github.com/zoobzio/tendo/cpu"
	"github.com/zoobzio/tendo/cuda"
	"github.com/zoobzio/tendo/models"
	"github.com/zoobzio/tendo/models/llama"
	"github.com/zoobzio/zyn"
)

const (
	// DeviceCUDA specifies NVIDIA GPU inference.
	DeviceCUDA = "cuda"
	// DeviceCPU specifies CPU inference.
	DeviceCPU = "cpu"
)

// Provider implements zyn.Provider for local LLM inference via tendo.
type Provider struct {
	model       *llama.Model
	tokenizer   *models.Tokenizer
	cudaBackend *cuda.Backend
	cpuBackend  *cpu.Backend
	template    Template
	config      Config
	name        string
}

// Config configures the tendo provider.
type Config struct {
	// ModelPath is the path to the model.safetensors file.
	ModelPath string
	// TokenizerPath is the path to the tokenizer.json file.
	TokenizerPath string
	// ModelConfig selects the model architecture configuration.
	// Supported: "smollm-135m", "smollm-360m", "qwen2-0.5b", "llama3.2-1b", "llama3.2-3b", "tinyllama"
	ModelConfig string
	// Device specifies the inference device: "cuda" or "cpu".
	Device string
	// Template specifies the chat template format.
	// If TemplateAuto (default), template is selected based on ModelConfig.
	Template TemplateType
	// MaxTokens is the maximum number of tokens to generate (default: 256).
	MaxTokens int
	// TopK limits sampling to top K tokens (0 = disabled).
	TopK int
	// TopP nucleus sampling threshold (0 = disabled).
	TopP float32
}

// New creates a new tendo provider, loading the model and tokenizer.
func New(config Config) (*Provider, error) {
	// Apply defaults
	if config.ModelConfig == "" {
		config.ModelConfig = "smollm-135m"
	}
	if config.Device == "" {
		config.Device = DeviceCPU
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 256
	}

	// Validate device
	device := strings.ToLower(config.Device)
	if device != DeviceCUDA && device != DeviceCPU {
		return nil, fmt.Errorf("tendo: invalid device %q (use %q or %q)", config.Device, DeviceCUDA, DeviceCPU)
	}
	if device == DeviceCUDA && !cuda.IsCUDAAvailable() {
		return nil, fmt.Errorf("tendo: CUDA requested but not available")
	}

	// Load tokenizer
	tokenizer, err := models.LoadTokenizer(config.TokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("tendo: load tokenizer: %w", err)
	}

	// Get model configuration
	modelCfg, err := getModelConfig(config.ModelConfig)
	if err != nil {
		_ = tokenizer.Close() //nolint:errcheck // best-effort cleanup
		return nil, err
	}

	// Load model
	var model *llama.Model
	var cudaBackend *cuda.Backend
	var cpuBackend *cpu.Backend

	switch device {
	case DeviceCUDA:
		cudaBackend = cuda.NewBackend()
		model, err = llama.LoadOn(config.ModelPath, modelCfg, cudaBackend)
	case DeviceCPU:
		cpuBackend = cpu.NewBackend()
		model, err = llama.Load(config.ModelPath, modelCfg)
	}

	if err != nil {
		_ = tokenizer.Close() //nolint:errcheck // best-effort cleanup
		return nil, fmt.Errorf("tendo: load model: %w", err)
	}

	// Select template
	var template Template
	if config.Template == TemplateAuto {
		template = templateForModel(config.ModelConfig)
	} else {
		template = templateForType(config.Template)
	}

	return &Provider{
		model:       model,
		tokenizer:   tokenizer,
		cudaBackend: cudaBackend,
		cpuBackend:  cpuBackend,
		template:    template,
		config:      config,
		name:        "tendo",
	}, nil
}

// Call sends messages to the local LLM and returns the response.
func (p *Provider) Call(ctx context.Context, messages []zyn.Message, temperature float32) (*zyn.ProviderResponse, error) {
	startTime := time.Now()

	// Emit started hook
	capitan.Info(ctx, zyn.ProviderCallStarted,
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.config.ModelConfig),
	)

	// Apply chat template
	prompt := p.template.Apply(messages)

	// Tokenize
	promptIDs := p.tokenizer.Encode(prompt, true)
	promptTokens := len(promptIDs)

	// Configure generation
	cfg := llama.GenerateConfig{
		MaxTokens:   p.config.MaxTokens,
		Temperature: temperature,
		TopK:        p.config.TopK,
		TopP:        p.config.TopP,
	}

	// Select backend for generation
	var backend llama.Backend
	if p.cudaBackend != nil {
		backend = p.cudaBackend
	} else {
		backend = p.cpuBackend
	}

	// Generate
	result, err := p.model.Generate(ctx, promptIDs, cfg, backend)
	if err != nil {
		duration := time.Since(startTime)
		capitan.Error(ctx, zyn.ProviderCallFailed,
			zyn.ProviderKey.Field(p.name),
			zyn.ModelKey.Field(p.config.ModelConfig),
			zyn.DurationMsKey.Field(int(duration.Milliseconds())),
			zyn.ErrorKey.Field(err.Error()),
		)
		return nil, fmt.Errorf("tendo: generate: %w", err)
	}

	// Decode generated tokens (excluding prompt)
	generatedIDs := result.TokenIDs[promptTokens:]
	content := p.tokenizer.Decode(generatedIDs, true)

	// Emit completed hook
	duration := time.Since(startTime)
	capitan.Info(ctx, zyn.ProviderCallCompleted,
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.config.ModelConfig),
		zyn.PromptTokensKey.Field(promptTokens),
		zyn.CompletionTokensKey.Field(result.NumTokens),
		zyn.TotalTokensKey.Field(promptTokens+result.NumTokens),
		zyn.DurationMsKey.Field(int(duration.Milliseconds())),
	)

	return &zyn.ProviderResponse{
		Content: content,
		Usage: zyn.TokenUsage{
			Prompt:     promptTokens,
			Completion: result.NumTokens,
			Total:      promptTokens + result.NumTokens,
		},
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return p.name
}

// Close releases provider resources.
func (p *Provider) Close() error {
	if p.tokenizer != nil {
		return p.tokenizer.Close()
	}
	return nil
}

// getModelConfig returns the llama.Config for a model name.
func getModelConfig(name string) (llama.Config, error) {
	switch strings.ToLower(name) {
	case "smollm-135m":
		return llama.ConfigSmolLM135M, nil
	case "smollm-360m":
		return llama.ConfigSmolLM360M, nil
	case "qwen2-0.5b":
		return llama.ConfigQwen2_0_5B, nil
	case "llama3.2-1b":
		return llama.ConfigLlama3_2_1B, nil
	case "llama3.2-3b":
		return llama.ConfigLlama3_2_3B, nil
	case "tinyllama":
		return llama.ConfigTinyLlama, nil
	default:
		return llama.Config{}, fmt.Errorf("tendo: unknown model config %q", name)
	}
}
