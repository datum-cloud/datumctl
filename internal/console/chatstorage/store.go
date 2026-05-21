// Package chatstorage persists AI chat conversations to ~/.datumctl/conversations/.
// Each conversation is a single JSON file; the store provides list, load, save,
// and last-conversation helpers used by the console AppModel.
package chatstorage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Message is one turn in a conversation.
type Message struct {
	Role      string    `json:"role"`    // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"at"`
}

// Conversation is the full record for one chat session.
type Conversation struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	OrgID     string    `json:"org_id,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	Messages  []Message `json:"messages"`
}

// Meta is a lightweight index entry for the conversation list.
type Meta struct {
	ID        string
	StartedAt time.Time
	UpdatedAt time.Time
	OrgID     string
	ProjectID string
	Preview   string // first user message, truncated
}

// Store manages conversation files under Dir.
type Store struct {
	Dir string
}

// DefaultDir returns ~/.datumctl/conversations.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".datumctl", "conversations"), nil
}

// NewStore creates a Store rooted at dir, creating the directory if needed.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create conversations dir: %w", err)
	}
	return &Store{Dir: dir}, nil
}

// NewConversation creates a new Conversation with a timestamp-based ID.
func NewConversation(orgID, projectID string) *Conversation {
	now := time.Now().UTC()
	id := now.Format("20060102-150405") + "-" + randomSuffix()
	return &Conversation{
		ID:        id,
		StartedAt: now,
		UpdatedAt: now,
		OrgID:     orgID,
		ProjectID: projectID,
	}
}

// AddMessage appends a message and updates UpdatedAt.
func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	})
	c.UpdatedAt = time.Now().UTC()
}

// Save writes the conversation to store as <ID>.json, overwriting any prior version.
func (s *Store) Save(c *Conversation) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal conversation: %w", err)
	}
	path := filepath.Join(s.Dir, c.ID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write conversation: %w", err)
	}
	return nil
}

// Load reads a conversation by ID.
func (s *Store) Load(id string) (*Conversation, error) {
	path := filepath.Join(s.Dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read conversation %s: %w", id, err)
	}
	var c Conversation
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse conversation %s: %w", id, err)
	}
	return &c, nil
}

// List returns metadata for all stored conversations, newest first.
func (s *Store) List() ([]*Meta, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	var metas []*Meta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		c, err := s.Load(id)
		if err != nil {
			continue // skip malformed files silently
		}
		m := &Meta{
			ID:        c.ID,
			StartedAt: c.StartedAt,
			UpdatedAt: c.UpdatedAt,
			OrgID:     c.OrgID,
			ProjectID: c.ProjectID,
		}
		for _, msg := range c.Messages {
			if msg.Role == "user" {
				preview := strings.ReplaceAll(msg.Content, "\n", " ")
				if len(preview) > 60 {
					preview = preview[:59] + "…"
				}
				m.Preview = preview
				break
			}
		}
		metas = append(metas, m)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.After(metas[j].UpdatedAt)
	})
	return metas, nil
}

// Delete removes a conversation by ID. Returns nil if the file does not exist.
func (s *Store) Delete(id string) error {
	path := filepath.Join(s.Dir, id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete conversation %s: %w", id, err)
	}
	return nil
}

// Last returns the most recently updated conversation, or nil if none exist.
func (s *Store) Last() (*Conversation, error) {
	metas, err := s.List()
	if err != nil || len(metas) == 0 {
		return nil, err
	}
	return s.Load(metas[0].ID)
}

// randomSuffix returns a short pseudo-random hex string for ID uniqueness.
func randomSuffix() string {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "000000"
	}
	defer f.Close()
	buf := make([]byte, 3)
	_, _ = f.Read(buf)
	return fmt.Sprintf("%x", buf)
}
