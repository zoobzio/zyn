package zyn

import (
	"context"
	"errors"
	"testing"

	"github.com/zoobzio/pipz"
)

func TestNewService(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		service := NewService[BinaryResponse](pipeline, "test", provider)

		if service == nil {
			t.Fatal("Expected service to be created")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		attempts := 0
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			attempts++
			if attempts < 2 {
				return req, errors.New("temporary failure")
			}
			return req, nil
		})

		service := NewService[BinaryResponse](pipeline, "test", provider)

		if service == nil {
			t.Fatal("Expected service to be created with failing pipeline")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		pipeline1 := pipz.Apply("stage1", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Prompt = &Prompt{Task: "modified", Input: "test", Schema: "{}"}
			return req, nil
		})
		pipeline2 := pipz.Apply("stage2", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})
		combined := pipz.NewSequence("combined", pipeline1, pipeline2)

		service := NewService[BinaryResponse](combined, "test", provider)

		if service.GetPipeline() == nil {
			t.Error("Service pipeline should be accessible")
		}
	})
}

func TestService_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})
		service := NewService[BinaryResponse](pipeline, "test", provider)

		retrieved := service.GetPipeline()
		if retrieved == nil {
			t.Error("GetPipeline returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		counter := 0
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			counter++
			return req, nil
		})
		service := NewService[BinaryResponse](pipeline, "test", provider)

		retrieved := service.GetPipeline()
		if retrieved == nil {
			t.Error("GetPipeline returned nil")
		}

		// Verify pipeline is functional
		ctx := context.Background()
		_, err := retrieved.Process(ctx, &SynapseRequest{})
		if err != nil {
			t.Errorf("Retrieved pipeline failed: %v", err)
		}
		if counter != 1 {
			t.Error("Pipeline should have been called")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Response = `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`
			return req, nil
		})
		service := NewService[BinaryResponse](pipeline, "test", provider)

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}

		_, err := service.GetPipeline().Process(ctx, &SynapseRequest{Prompt: prompt})
		if err != nil {
			t.Errorf("Pipeline processing failed: %v", err)
		}
	})
}

func TestService_Execute(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Response = `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`
			return req, nil
		})
		service := NewService[BinaryResponse](pipeline, "test", provider)

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		response, err := service.Execute(ctx, prompt, 0.5)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if !response.Decision {
			t.Error("Expected decision to be true")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		attempts := 0
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			attempts++
			if attempts < 2 {
				return req, errors.New("temporary failure")
			}
			req.Response = `{"decision": true, "confidence": 0.8, "reasoning": ["test"]}`
			return req, nil
		})
		service := NewService[BinaryResponse](pipeline, "test", provider)

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		_, err := service.Execute(ctx, prompt, 0.5)
		if err == nil {
			t.Error("Expected error from failing pipeline")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		modifyPipeline := pipz.Apply("modify", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Prompt.Task = "modified task"
			return req, nil
		})
		executePipeline := pipz.Apply("execute", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Response = `{"decision": false, "confidence": 0.7, "reasoning": ["modified"]}`
			return req, nil
		})
		combined := pipz.NewSequence("combined", modifyPipeline, executePipeline)
		service := NewService[BinaryResponse](combined, "test", provider)

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		response, err := service.Execute(ctx, prompt, 0.5)
		if err != nil {
			t.Fatalf("Execute with chained pipeline failed: %v", err)
		}
		if response.Decision {
			t.Error("Expected decision to be false from chained pipeline")
		}
	})
}
