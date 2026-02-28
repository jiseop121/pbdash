package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Superuser struct {
	DBAlias  string `json:"dbAlias"`
	Alias    string `json:"alias"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SuperuserStore struct {
	path string
}

func NewSuperuserStore(dataDir string) *SuperuserStore {
	return &SuperuserStore{path: filepath.Join(dataDir, "superusers.json")}
}

func (s *SuperuserStore) Add(dbAlias, alias, email, password string) error {
	if strings.TrimSpace(dbAlias) == "" || strings.TrimSpace(alias) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
		return NewValidationError("--db, --alias, --email, and --password are required")
	}

	items, err := s.readAll()
	if err != nil {
		return err
	}
	for _, it := range items {
		if strings.EqualFold(it.DBAlias, dbAlias) && strings.EqualFold(it.Alias, alias) {
			return NewValidationError(fmt.Sprintf("superuser alias %q already exists for db %q", alias, dbAlias))
		}
	}

	items = append(items, Superuser{DBAlias: dbAlias, Alias: alias, Email: email, Password: password})
	return s.writeAll(items)
}

func (s *SuperuserStore) ListByDB(dbAlias string) ([]Superuser, error) {
	items, err := s.readAll()
	if err != nil {
		return nil, err
	}
	filtered := make([]Superuser, 0)
	for _, it := range items {
		if strings.EqualFold(it.DBAlias, dbAlias) {
			filtered = append(filtered, it)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Alias) < strings.ToLower(filtered[j].Alias)
	})
	return filtered, nil
}

func (s *SuperuserStore) Remove(dbAlias, alias string) error {
	items, err := s.readAll()
	if err != nil {
		return err
	}
	filtered := make([]Superuser, 0, len(items))
	removed := false
	for _, it := range items {
		if strings.EqualFold(it.DBAlias, dbAlias) && strings.EqualFold(it.Alias, alias) {
			removed = true
			continue
		}
		filtered = append(filtered, it)
	}
	if !removed {
		return NewValidationError(fmt.Sprintf("superuser alias %q is not configured for db %q", alias, dbAlias))
	}
	return s.writeAll(filtered)
}

func (s *SuperuserStore) Find(dbAlias, alias string) (Superuser, bool, error) {
	items, err := s.readAll()
	if err != nil {
		return Superuser{}, false, err
	}
	for _, it := range items {
		if strings.EqualFold(it.DBAlias, dbAlias) && strings.EqualFold(it.Alias, alias) {
			return it, true, nil
		}
	}
	return Superuser{}, false, nil
}

func (s *SuperuserStore) readAll() ([]Superuser, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Superuser{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []Superuser{}, nil
	}
	var items []Superuser
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SuperuserStore) writeAll(items []Superuser) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}
