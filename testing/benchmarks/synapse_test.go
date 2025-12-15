package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/zoobzio/zyn"
	zynt "github.com/zoobzio/zyn/testing"
)

// Sink variables to prevent compiler optimizations.
var (
	sinkBool   bool
	sinkString string
	sinkError  error
)

func BenchmarkSynapse_Creation(b *testing.B) {
	provider := zyn.NewMockProvider()

	b.Run("Binary", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, err := zyn.Binary("Is this valid?", provider)
			sinkError = err
			_ = s
		}
	})

	b.Run("Classification", func(b *testing.B) {
		categories := []string{"spam", "ham", "urgent", "newsletter"}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, err := zyn.Classification("Classify this", categories, provider)
			sinkError = err
			_ = s
		}
	})

	b.Run("Transform", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, err := zyn.Transform("Summarize this", provider)
			sinkError = err
			_ = s
		}
	})

	b.Run("Sentiment", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, err := zyn.Sentiment("Analyze sentiment", provider)
			sinkError = err
			_ = s
		}
	})

	b.Run("Ranking", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, err := zyn.Ranking("by importance", provider)
			sinkError = err
			_ = s
		}
	})
}

func BenchmarkSynapse_CreationWithOptions(b *testing.B) {
	provider := zyn.NewMockProvider()

	b.Run("NoOptions", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, _ := zyn.Binary("question", provider)
			_ = s
		}
	})

	b.Run("WithRetry", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, _ := zyn.Binary("question", provider, zyn.WithRetry(3))
			_ = s
		}
	})

	b.Run("WithTimeout", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, _ := zyn.Binary("question", provider, zyn.WithTimeout(30*time.Second))
			_ = s
		}
	})

	b.Run("WithMultipleOptions", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s, _ := zyn.Binary("question", provider,
				zyn.WithRetry(3),
				zyn.WithTimeout(30*time.Second),
				zyn.WithCircuitBreaker(5, 30*time.Second),
			)
			_ = s
		}
	})
}

func BenchmarkSynapse_Fire(b *testing.B) {
	provider := zyn.NewMockProvider()
	ctx := context.Background()

	b.Run("Binary", func(b *testing.B) {
		synapse, _ := zyn.Binary("Is this valid?", provider)
		session := zyn.NewSession()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session.Clear()
			result, err := synapse.Fire(ctx, session, "test@example.com")
			sinkBool = result
			sinkError = err
		}
	})

	b.Run("Classification", func(b *testing.B) {
		synapse, _ := zyn.Classification("Classify", []string{"a", "b", "c"}, provider)
		session := zyn.NewSession()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session.Clear()
			result, err := synapse.Fire(ctx, session, "input text")
			sinkString = result
			sinkError = err
		}
	})

	b.Run("Transform", func(b *testing.B) {
		synapse, _ := zyn.Transform("Summarize", provider)
		session := zyn.NewSession()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session.Clear()
			result, err := synapse.Fire(ctx, session, "input text")
			sinkString = result
			sinkError = err
		}
	})

	b.Run("Sentiment", func(b *testing.B) {
		synapse, _ := zyn.Sentiment("Analyze", provider)
		session := zyn.NewSession()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session.Clear()
			result, err := synapse.Fire(ctx, session, "I love this!")
			sinkString = result
			sinkError = err
		}
	})
}

func BenchmarkSynapse_FireWithSession(b *testing.B) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
	)
	synapse, _ := zyn.Binary("question", provider)
	ctx := context.Background()

	b.Run("EmptySession", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session := zyn.NewSession()
			provider.Reset()
			result, err := synapse.Fire(ctx, session, "input")
			sinkBool = result
			sinkError = err
		}
	})

	b.Run("SessionWith10Messages", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session := zyn.NewSession()
			// Pre-populate with 10 messages
			for j := 0; j < 5; j++ {
				session.Append(zyn.RoleUser, "user message")
				session.Append(zyn.RoleAssistant, "assistant message")
			}
			provider.Reset()
			result, err := synapse.Fire(ctx, session, "input")
			sinkBool = result
			sinkError = err
		}
	})

	b.Run("SessionWith50Messages", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session := zyn.NewSession()
			// Pre-populate with 50 messages
			for j := 0; j < 25; j++ {
				session.Append(zyn.RoleUser, "user message")
				session.Append(zyn.RoleAssistant, "assistant message")
			}
			provider.Reset()
			result, err := synapse.Fire(ctx, session, "input")
			sinkBool = result
			sinkError = err
		}
	})
}

func BenchmarkSession_Operations(b *testing.B) {
	b.Run("NewSession", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := zyn.NewSession()
			_ = s
		}
	})

	b.Run("Append", func(b *testing.B) {
		session := zyn.NewSession()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			session.Append(zyn.RoleUser, "message content")
		}
	})

	b.Run("Messages", func(b *testing.B) {
		session := zyn.NewSession()
		for i := 0; i < 20; i++ {
			session.Append(zyn.RoleUser, "user message")
			session.Append(zyn.RoleAssistant, "assistant message")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msgs := session.Messages()
			_ = msgs
		}
	})

	b.Run("Clear", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			session := zyn.NewSession()
			for j := 0; j < 10; j++ {
				session.Append(zyn.RoleUser, "message")
			}
			session.Clear()
		}
	})

	b.Run("Prune", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			session := zyn.NewSession()
			for j := 0; j < 20; j++ {
				session.Append(zyn.RoleUser, "user message")
				session.Append(zyn.RoleAssistant, "assistant message")
			}
			_ = session.Prune(5)
		}
	})

	b.Run("Truncate", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			session := zyn.NewSession()
			for j := 0; j < 20; j++ {
				session.Append(zyn.RoleUser, "user message")
				session.Append(zyn.RoleAssistant, "assistant message")
			}
			_ = session.Truncate(2, 2)
		}
	})
}

func BenchmarkResponseBuilder(b *testing.B) {
	b.Run("Binary", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			response := zynt.NewResponseBuilder().
				WithDecision(true).
				WithConfidence(0.95).
				WithReasoning("reason1", "reason2").
				Build()
			sinkString = response
		}
	})

	b.Run("Classification", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			response := zynt.NewResponseBuilder().
				WithPrimary("category").
				WithSecondary("subcategory").
				WithConfidence(0.85).
				WithReasoning("reason").
				Build()
			sinkString = response
		}
	})

	b.Run("Sentiment", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			response := zynt.NewResponseBuilder().
				WithOverall("positive").
				WithConfidence(0.9).
				WithScores(0.7, 0.1, 0.2).
				WithEmotions("joy", "satisfaction").
				WithAspects(map[string]string{"service": "positive"}).
				WithReasoning("positive tone").
				Build()
			sinkString = response
		}
	})
}

func BenchmarkCallRecorder(b *testing.B) {
	inner := zyn.NewMockProvider()
	ctx := context.Background()
	messages := []zyn.Message{{Role: zyn.RoleUser, Content: "test"}}

	b.Run("RecordCall", func(b *testing.B) {
		recorder := zynt.NewCallRecorder(inner)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = recorder.Call(ctx, messages, 0.5)
		}
	})

	b.Run("GetCalls", func(b *testing.B) {
		recorder := zynt.NewCallRecorder(inner)
		for i := 0; i < 100; i++ {
			_, _ = recorder.Call(ctx, messages, 0.5)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			calls := recorder.Calls()
			_ = calls
		}
	})
}

func BenchmarkUsageAccumulator(b *testing.B) {
	usage := &zyn.TokenUsage{Prompt: 100, Completion: 50, Total: 150}

	b.Run("AddUsage", func(b *testing.B) {
		acc := zynt.NewUsageAccumulator()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			acc.AddUsage(usage)
		}
	})

	b.Run("GetTotals", func(b *testing.B) {
		acc := zynt.NewUsageAccumulator()
		for i := 0; i < 1000; i++ {
			acc.AddUsage(usage)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = acc.TotalTokens()
			_ = acc.PromptTokens()
			_ = acc.CompletionTokens()
			_ = acc.CallCount()
		}
	})
}

// Concurrent benchmarks measure performance under parallel load.

func BenchmarkConcurrent_SynapseFire(b *testing.B) {
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("question", provider)
	ctx := context.Background()

	b.Run("Parallel_4", func(b *testing.B) {
		b.SetParallelism(4)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			session := zyn.NewSession()
			for pb.Next() {
				session.Clear()
				result, err := synapse.Fire(ctx, session, "input")
				sinkBool = result
				sinkError = err
			}
		})
	})

	b.Run("Parallel_16", func(b *testing.B) {
		b.SetParallelism(16)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			session := zyn.NewSession()
			for pb.Next() {
				session.Clear()
				result, err := synapse.Fire(ctx, session, "input")
				sinkBool = result
				sinkError = err
			}
		})
	})
}

func BenchmarkConcurrent_SharedSession(b *testing.B) {
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	b.Run("Parallel_4", func(b *testing.B) {
		b.SetParallelism(4)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				result, err := synapse.Fire(ctx, session, "input")
				sinkBool = result
				sinkError = err
			}
		})
	})
}

func BenchmarkConcurrent_SessionAppend(b *testing.B) {
	b.Run("Parallel_4", func(b *testing.B) {
		session := zyn.NewSession()
		b.SetParallelism(4)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				session.Append(zyn.RoleUser, "concurrent message")
			}
		})
	})

	b.Run("Parallel_16", func(b *testing.B) {
		session := zyn.NewSession()
		b.SetParallelism(16)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				session.Append(zyn.RoleUser, "concurrent message")
			}
		})
	})
}

func BenchmarkConcurrent_SessionMessages(b *testing.B) {
	session := zyn.NewSession()
	for i := 0; i < 100; i++ {
		session.Append(zyn.RoleUser, "user message")
		session.Append(zyn.RoleAssistant, "assistant message")
	}

	b.Run("Parallel_4", func(b *testing.B) {
		b.SetParallelism(4)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				msgs := session.Messages()
				_ = msgs
			}
		})
	})
}

func BenchmarkConcurrent_CallRecorder(b *testing.B) {
	inner := zyn.NewMockProvider()
	recorder := zynt.NewCallRecorder(inner)
	ctx := context.Background()
	messages := []zyn.Message{{Role: zyn.RoleUser, Content: "test"}}

	b.Run("Parallel_Record", func(b *testing.B) {
		b.SetParallelism(8)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = recorder.Call(ctx, messages, 0.5)
			}
		})
	})
}

func BenchmarkConcurrent_UsageAccumulator(b *testing.B) {
	acc := zynt.NewUsageAccumulator()
	usage := &zyn.TokenUsage{Prompt: 100, Completion: 50, Total: 150}

	b.Run("Parallel_Add", func(b *testing.B) {
		b.SetParallelism(8)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				acc.AddUsage(usage)
			}
		})
	})
}

func BenchmarkConcurrent_MultipleSynapseTypes(b *testing.B) {
	provider := zyn.NewMockProvider()
	binary, _ := zyn.Binary("question", provider)
	classify, _ := zyn.Classification("type", []string{"a", "b"}, provider)
	sentiment, _ := zyn.Sentiment("sentiment", provider)
	ctx := context.Background()

	b.Run("Parallel_Mixed", func(b *testing.B) {
		b.SetParallelism(12) // 4 per synapse type
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			session := zyn.NewSession()
			i := 0
			for pb.Next() {
				session.Clear()
				switch i % 3 {
				case 0:
					result, err := binary.Fire(ctx, session, "input")
					sinkBool = result
					sinkError = err
				case 1:
					result, err := classify.Fire(ctx, session, "input")
					sinkString = result
					sinkError = err
				case 2:
					result, err := sentiment.Fire(ctx, session, "input")
					sinkString = result
					sinkError = err
				}
				i++
			}
		})
	})
}
