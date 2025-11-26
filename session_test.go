package zyn

import (
	"testing"
)

func TestNewSession(t *testing.T) {
	session := NewSession()

	if session == nil {
		t.Fatal("NewSession returned nil")
	}
	if session.ID() == "" {
		t.Error("Session ID should not be empty")
	}
	if session.Len() != 0 {
		t.Errorf("New session should have 0 messages, got %d", session.Len())
	}
}

func TestSession_ID(t *testing.T) {
	session1 := NewSession()
	session2 := NewSession()

	if session1.ID() == session2.ID() {
		t.Error("Different sessions should have different IDs")
	}

	// ID should be consistent
	id1 := session1.ID()
	id2 := session1.ID()
	if id1 != id2 {
		t.Error("Session ID should be consistent across calls")
	}
}

func TestSession_Len(t *testing.T) {
	session := NewSession()

	if session.Len() != 0 {
		t.Errorf("Expected 0, got %d", session.Len())
	}

	session.Append(RoleUser, "hello")
	if session.Len() != 1 {
		t.Errorf("Expected 1, got %d", session.Len())
	}

	session.Append(RoleAssistant, "hi")
	if session.Len() != 2 {
		t.Errorf("Expected 2, got %d", session.Len())
	}
}

func TestSession_At(t *testing.T) {
	t.Run("valid index", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "first")
		session.Append(RoleAssistant, "second")

		msg, err := session.At(0)
		if err != nil {
			t.Fatalf("At(0) failed: %v", err)
		}
		if msg.Content != "first" {
			t.Errorf("Expected 'first', got '%s'", msg.Content)
		}

		msg, err = session.At(1)
		if err != nil {
			t.Fatalf("At(1) failed: %v", err)
		}
		if msg.Content != "second" {
			t.Errorf("Expected 'second', got '%s'", msg.Content)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		_, err := session.At(-1)
		if err == nil {
			t.Error("Expected error for negative index")
		}

		_, err = session.At(1)
		if err == nil {
			t.Error("Expected error for index >= len")
		}

		_, err = session.At(100)
		if err == nil {
			t.Error("Expected error for large index")
		}
	})

	t.Run("empty session", func(t *testing.T) {
		session := NewSession()

		_, err := session.At(0)
		if err == nil {
			t.Error("Expected error for empty session")
		}
	})
}

func TestSession_Remove(t *testing.T) {
	t.Run("valid removal", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "first")
		session.Append(RoleAssistant, "second")
		session.Append(RoleUser, "third")

		err := session.Remove(1)
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		if session.Len() != 2 {
			t.Errorf("Expected 2 messages, got %d", session.Len())
		}

		msg, _ := session.At(1)
		if msg.Content != "third" {
			t.Errorf("Expected 'third' at index 1, got '%s'", msg.Content)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		err := session.Remove(-1)
		if err == nil {
			t.Error("Expected error for negative index")
		}

		err = session.Remove(1)
		if err == nil {
			t.Error("Expected error for index >= len")
		}
	})

	t.Run("remove all", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "only")

		err := session.Remove(0)
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		if session.Len() != 0 {
			t.Error("Session should be empty")
		}
	})
}

func TestSession_Replace(t *testing.T) {
	t.Run("valid replacement", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "original")

		err := session.Replace(0, Message{Role: RoleUser, Content: "replaced"})
		if err != nil {
			t.Fatalf("Replace failed: %v", err)
		}

		msg, _ := session.At(0)
		if msg.Content != "replaced" {
			t.Errorf("Expected 'replaced', got '%s'", msg.Content)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		err := session.Replace(-1, Message{Role: RoleUser, Content: "new"})
		if err == nil {
			t.Error("Expected error for negative index")
		}

		err = session.Replace(1, Message{Role: RoleUser, Content: "new"})
		if err == nil {
			t.Error("Expected error for index >= len")
		}
	})
}

func TestSession_Truncate(t *testing.T) {
	t.Run("normal truncation", func(t *testing.T) {
		session := NewSession()
		for i := 0; i < 10; i++ {
			session.Append(RoleUser, string(rune('a'+i)))
		}

		err := session.Truncate(2, 2)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}

		if session.Len() != 4 {
			t.Errorf("Expected 4 messages, got %d", session.Len())
		}

		// Check first two
		msg, _ := session.At(0)
		if msg.Content != "a" {
			t.Errorf("Expected 'a', got '%s'", msg.Content)
		}
		msg, _ = session.At(1)
		if msg.Content != "b" {
			t.Errorf("Expected 'b', got '%s'", msg.Content)
		}

		// Check last two
		msg, _ = session.At(2)
		if msg.Content != "i" {
			t.Errorf("Expected 'i', got '%s'", msg.Content)
		}
		msg, _ = session.At(3)
		if msg.Content != "j" {
			t.Errorf("Expected 'j', got '%s'", msg.Content)
		}
	})

	t.Run("keepFirst + keepLast >= len", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "a")
		session.Append(RoleAssistant, "b")
		session.Append(RoleUser, "c")

		err := session.Truncate(2, 2)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}

		// Should keep all messages
		if session.Len() != 3 {
			t.Errorf("Expected 3 messages (no change), got %d", session.Len())
		}
	})

	t.Run("negative values", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		err := session.Truncate(-1, 1)
		if err == nil {
			t.Error("Expected error for negative keepFirst")
		}

		err = session.Truncate(1, -1)
		if err == nil {
			t.Error("Expected error for negative keepLast")
		}
	})

	t.Run("keep only first", func(t *testing.T) {
		session := NewSession()
		for i := 0; i < 5; i++ {
			session.Append(RoleUser, string(rune('a'+i)))
		}

		err := session.Truncate(2, 0)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}

		if session.Len() != 2 {
			t.Errorf("Expected 2 messages, got %d", session.Len())
		}
	})

	t.Run("keep only last", func(t *testing.T) {
		session := NewSession()
		for i := 0; i < 5; i++ {
			session.Append(RoleUser, string(rune('a'+i)))
		}

		err := session.Truncate(0, 2)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}

		if session.Len() != 2 {
			t.Errorf("Expected 2 messages, got %d", session.Len())
		}

		msg, _ := session.At(0)
		if msg.Content != "d" {
			t.Errorf("Expected 'd', got '%s'", msg.Content)
		}
	})
}

func TestSession_Insert(t *testing.T) {
	t.Run("insert at beginning", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "second")

		err := session.Insert(0, Message{Role: "system", Content: "first"})
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if session.Len() != 2 {
			t.Errorf("Expected 2 messages, got %d", session.Len())
		}

		msg, _ := session.At(0)
		if msg.Content != "first" {
			t.Errorf("Expected 'first', got '%s'", msg.Content)
		}
	})

	t.Run("insert at end", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "first")

		err := session.Insert(1, Message{Role: RoleAssistant, Content: "second"})
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		msg, _ := session.At(1)
		if msg.Content != "second" {
			t.Errorf("Expected 'second', got '%s'", msg.Content)
		}
	})

	t.Run("insert in middle", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "first")
		session.Append(RoleUser, "third")

		err := session.Insert(1, Message{Role: RoleAssistant, Content: "second"})
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if session.Len() != 3 {
			t.Errorf("Expected 3 messages, got %d", session.Len())
		}

		msg, _ := session.At(1)
		if msg.Content != "second" {
			t.Errorf("Expected 'second', got '%s'", msg.Content)
		}

		msg, _ = session.At(2)
		if msg.Content != "third" {
			t.Errorf("Expected 'third', got '%s'", msg.Content)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		err := session.Insert(-1, Message{Role: RoleUser, Content: "new"})
		if err == nil {
			t.Error("Expected error for negative index")
		}

		err = session.Insert(5, Message{Role: RoleUser, Content: "new"})
		if err == nil {
			t.Error("Expected error for index > len")
		}
	})

	t.Run("insert into empty", func(t *testing.T) {
		session := NewSession()

		err := session.Insert(0, Message{Role: "system", Content: "first"})
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if session.Len() != 1 {
			t.Errorf("Expected 1 message, got %d", session.Len())
		}
	})
}

func TestSession_SetMessages(t *testing.T) {
	t.Run("replace all", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "old1")
		session.Append(RoleAssistant, "old2")

		newMsgs := []Message{
			{Role: "system", Content: "new1"},
			{Role: RoleUser, Content: "new2"},
			{Role: RoleAssistant, Content: "new3"},
		}

		session.SetMessages(newMsgs)

		if session.Len() != 3 {
			t.Errorf("Expected 3 messages, got %d", session.Len())
		}

		msg, _ := session.At(0)
		if msg.Content != "new1" {
			t.Errorf("Expected 'new1', got '%s'", msg.Content)
		}
	})

	t.Run("set empty", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		session.SetMessages([]Message{})

		if session.Len() != 0 {
			t.Errorf("Expected 0 messages, got %d", session.Len())
		}
	})

	t.Run("isolation from source", func(t *testing.T) {
		session := NewSession()

		msgs := []Message{{Role: RoleUser, Content: "original"}}
		session.SetMessages(msgs)

		// Modify source slice
		msgs[0].Content = "modified"

		// Session should be unaffected
		msg, _ := session.At(0)
		if msg.Content != "original" {
			t.Error("SetMessages should copy, not reference")
		}
	})
}

func TestSession_LastUsage(t *testing.T) {
	t.Run("initially nil", func(t *testing.T) {
		session := NewSession()

		usage := session.LastUsage()
		if usage != nil {
			t.Error("LastUsage should be nil for new session")
		}
	})

	t.Run("after SetUsage", func(t *testing.T) {
		session := NewSession()

		session.SetUsage(&TokenUsage{
			Prompt:     100,
			Completion: 50,
			Total:      150,
		})

		usage := session.LastUsage()
		if usage == nil {
			t.Fatal("LastUsage should not be nil after SetUsage")
		}
		if usage.Prompt != 100 {
			t.Errorf("Expected Prompt=100, got %d", usage.Prompt)
		}
		if usage.Completion != 50 {
			t.Errorf("Expected Completion=50, got %d", usage.Completion)
		}
		if usage.Total != 150 {
			t.Errorf("Expected Total=150, got %d", usage.Total)
		}
	})

	t.Run("returns copy", func(t *testing.T) {
		session := NewSession()
		session.SetUsage(&TokenUsage{Prompt: 100, Completion: 50, Total: 150})

		usage1 := session.LastUsage()
		usage1.Prompt = 999

		usage2 := session.LastUsage()
		if usage2.Prompt != 100 {
			t.Error("LastUsage should return a copy")
		}
	})

	t.Run("SetUsage with nil", func(t *testing.T) {
		session := NewSession()
		session.SetUsage(&TokenUsage{Prompt: 100, Completion: 50, Total: 150})

		session.SetUsage(nil)

		// Should retain previous value (nil doesn't overwrite)
		usage := session.LastUsage()
		if usage == nil {
			t.Error("SetUsage(nil) should not clear existing usage")
		}
	})

	t.Run("overwrite usage", func(t *testing.T) {
		session := NewSession()
		session.SetUsage(&TokenUsage{Prompt: 100, Completion: 50, Total: 150})
		session.SetUsage(&TokenUsage{Prompt: 200, Completion: 100, Total: 300})

		usage := session.LastUsage()
		if usage.Total != 300 {
			t.Errorf("Expected Total=300, got %d", usage.Total)
		}
	})
}

func TestSession_Prune(t *testing.T) {
	t.Run("prune pairs", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "u1")
		session.Append(RoleAssistant, "a1")
		session.Append(RoleUser, "u2")
		session.Append(RoleAssistant, "a2")

		err := session.Prune(1)
		if err != nil {
			t.Fatalf("Prune failed: %v", err)
		}

		if session.Len() != 2 {
			t.Errorf("Expected 2 messages, got %d", session.Len())
		}

		msg, _ := session.At(1)
		if msg.Content != "a1" {
			t.Errorf("Expected 'a1', got '%s'", msg.Content)
		}
	})

	t.Run("prune more than exists", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "u1")
		session.Append(RoleAssistant, "a1")

		err := session.Prune(10)
		if err != nil {
			t.Fatalf("Prune failed: %v", err)
		}

		if session.Len() != 0 {
			t.Errorf("Expected 0 messages, got %d", session.Len())
		}
	})

	t.Run("negative prune", func(t *testing.T) {
		session := NewSession()

		err := session.Prune(-1)
		if err == nil {
			t.Error("Expected error for negative prune count")
		}
	})
}

func TestSession_Clear(t *testing.T) {
	session := NewSession()
	session.Append(RoleUser, "u1")
	session.Append(RoleAssistant, "a1")

	session.Clear()

	if session.Len() != 0 {
		t.Errorf("Expected 0 messages after Clear, got %d", session.Len())
	}

	// ID should remain the same
	if session.ID() == "" {
		t.Error("ID should not be empty after Clear")
	}
}

func TestSession_Messages(t *testing.T) {
	t.Run("returns copy", func(t *testing.T) {
		session := NewSession()
		session.Append(RoleUser, "test")

		msgs := session.Messages()
		msgs[0].Content = "modified"

		// Original should be unchanged
		msg, _ := session.At(0)
		if msg.Content != "test" {
			t.Error("Messages should return a copy")
		}
	})

	t.Run("empty session", func(t *testing.T) {
		session := NewSession()

		msgs := session.Messages()
		if len(msgs) != 0 {
			t.Errorf("Expected empty slice, got %d messages", len(msgs))
		}
	})
}
