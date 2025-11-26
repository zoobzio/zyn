package zyn

import (
	"fmt"
	"slices"
	"sync"

	"github.com/google/uuid"
)

// Role constants for message types.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a single message in a conversation.
// Messages are exchanged between the user and the assistant (LLM).
type Message struct {
	Role    string // RoleUser or RoleAssistant
	Content string // The message content
}

// Session manages conversation state across multiple synapse calls.
// It stores message history and enables multi-turn conversations with
// automatic prompt caching support from providers.
//
// Sessions are safe for concurrent use by multiple goroutines.
type Session struct {
	id        string
	messages  []Message
	lastUsage *TokenUsage
	mu        sync.RWMutex
}

// NewSession creates a new conversation session with a unique ID.
// Each session maintains its own message history independent of other sessions.
//
// Example:
//
//	session := zyn.NewSession()
//	result1, _ := synapse.Fire(ctx, session, input1)
//	result2, _ := synapse.Fire(ctx, session, input2) // Sees input1 context
func NewSession() *Session {
	return &Session{
		id:       uuid.New().String(),
		messages: make([]Message, 0),
	}
}

// ID returns the unique identifier for this session.
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// Messages returns a copy of all messages in the session.
// The returned slice is a copy and safe to modify without affecting the session.
func (s *Session) Messages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	messages := make([]Message, len(s.messages))
	copy(messages, s.messages)
	return messages
}

// Append adds a new message to the session.
// Role should be RoleUser or RoleAssistant.
// Content is the message text.
//
// This method is typically called internally by synapses after successful
// LLM calls, but can be used directly for manual session management.
func (s *Session) Append(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, Message{
		Role:    role,
		Content: content,
	})
}

// Clear removes all messages from the session.
// Use this when you want to start a fresh conversation in the same session.
//
// Example:
//
//	if errors.Is(err, zyn.ErrContextLength) {
//	    session.Clear()
//	    result, err = synapse.Fire(ctx, session, input) // Retry with empty context
//	}
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = make([]Message, 0)
}

// Prune removes the last n message pairs (user + assistant) from the session.
// Each pair consists of 2 messages, so n=1 removes 2 messages.
// If n would remove more messages than exist, all messages are removed.
//
// This is useful for managing context window size while preserving
// recent conversation history.
//
// Example:
//
//	session.Prune(2) // Remove last 2 exchanges (4 messages)
func (s *Session) Prune(n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n < 0 {
		return fmt.Errorf("prune count must be non-negative, got %d", n)
	}

	// Each pair is 2 messages
	messagesToRemove := n * 2

	if messagesToRemove >= len(s.messages) {
		// Remove all messages
		s.messages = make([]Message, 0)
		return nil
	}

	// Keep messages up to the prune point
	keepCount := len(s.messages) - messagesToRemove
	s.messages = s.messages[:keepCount]

	return nil
}

// Len returns the number of messages in the session.
func (s *Session) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

// LastUsage returns the token usage from the most recent provider call.
// Returns nil if no calls have been made yet.
func (s *Session) LastUsage() *TokenUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastUsage == nil {
		return nil
	}
	// Return a copy
	usage := *s.lastUsage
	return &usage
}

// SetUsage updates the session's last usage statistics.
// This is called internally by the service after successful provider calls.
func (s *Session) SetUsage(usage *TokenUsage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if usage != nil {
		u := *usage
		s.lastUsage = &u
	}
}

// At returns the message at the given index.
// Returns an error if the index is out of bounds.
func (s *Session) At(index int) (Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.messages) {
		return Message{}, fmt.Errorf("index %d out of bounds (len=%d)", index, len(s.messages))
	}
	return s.messages[index], nil
}

// Remove deletes the message at the given index.
// Returns an error if the index is out of bounds.
func (s *Session) Remove(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.messages) {
		return fmt.Errorf("index %d out of bounds (len=%d)", index, len(s.messages))
	}

	// Clone before mutation to prevent aliasing issues with any external slice references
	s.messages = slices.Clone(s.messages)
	s.messages = slices.Delete(s.messages, index, index+1)
	return nil
}

// Replace swaps the message at the given index with a new message.
// Returns an error if the index is out of bounds.
func (s *Session) Replace(index int, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.messages) {
		return fmt.Errorf("index %d out of bounds (len=%d)", index, len(s.messages))
	}

	s.messages[index] = msg
	return nil
}

// Truncate keeps only the first keepFirst messages and the last keepLast messages,
// removing everything in between.
// Returns an error if the parameters are invalid.
func (s *Session) Truncate(keepFirst, keepLast int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if keepFirst < 0 || keepLast < 0 {
		return fmt.Errorf("keepFirst and keepLast must be non-negative")
	}

	total := len(s.messages)
	if keepFirst+keepLast >= total {
		// Nothing to remove
		return nil
	}

	// Build new slice: first N + last M
	newMessages := make([]Message, 0, keepFirst+keepLast)
	newMessages = append(newMessages, s.messages[:keepFirst]...)
	newMessages = append(newMessages, s.messages[total-keepLast:]...)
	s.messages = newMessages

	return nil
}

// Insert adds a message at the given index, shifting subsequent messages.
// If index equals Len(), the message is appended.
// Returns an error if the index is out of bounds.
func (s *Session) Insert(index int, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index > len(s.messages) {
		return fmt.Errorf("index %d out of bounds (len=%d)", index, len(s.messages))
	}

	// Insert at index using slices.Insert for efficiency
	s.messages = slices.Insert(s.messages, index, msg)
	return nil
}

// SetMessages replaces the entire message history.
// This is useful for external context management strategies.
func (s *Session) SetMessages(msgs []Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy
	s.messages = make([]Message, len(msgs))
	copy(s.messages, msgs)
}
