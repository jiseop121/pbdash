package storage

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Context struct {
	DBAlias        string `json:"dbAlias"`
	SuperuserAlias string `json:"superuserAlias,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
}

type ContextStore struct {
	path string
	now  func() time.Time
}

func NewContextStore(dataDir string) *ContextStore {
	return &ContextStore{
		path: filepath.Join(dataDir, "context.json"),
		now:  time.Now,
	}
}

func (s *ContextStore) Load() (Context, bool, error) {
	var saved Context
	found, err := readJSONFile(s.path, &saved)
	if err != nil {
		return Context{}, false, err
	}
	if !found || strings.TrimSpace(saved.DBAlias) == "" {
		return Context{}, false, nil
	}
	return saved, true, nil
}

func (s *ContextStore) Save(ctx Context) error {
	if strings.TrimSpace(ctx.DBAlias) == "" {
		return NewValidationError("context db alias is required")
	}
	if strings.TrimSpace(ctx.UpdatedAt) == "" {
		ctx.UpdatedAt = s.now().UTC().Format(time.RFC3339)
	}
	return writeJSONFile(s.path, ctx)
}

func (s *ContextStore) Clear() error {
	err := os.Remove(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
