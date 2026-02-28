package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DB struct {
	Alias   string `json:"alias"`
	BaseURL string `json:"baseUrl"`
}

type DBStore struct {
	path string
}

func NewDBStore(dataDir string) *DBStore {
	return &DBStore{path: filepath.Join(dataDir, "dbs.json")}
}

func (s *DBStore) Add(alias, baseURL string) error {
	if strings.TrimSpace(alias) == "" {
		return NewValidationError("db alias is required")
	}
	if err := validateBaseURL(baseURL); err != nil {
		return err
	}

	items, err := s.readAll()
	if err != nil {
		return err
	}
	for _, it := range items {
		if strings.EqualFold(it.Alias, alias) {
			return NewValidationError(fmt.Sprintf("db alias %q already exists", alias))
		}
	}

	items = append(items, DB{Alias: alias, BaseURL: baseURL})
	return s.writeAll(items)
}

func (s *DBStore) List() ([]DB, error) {
	items, err := s.readAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Alias) < strings.ToLower(items[j].Alias)
	})
	return items, nil
}

func (s *DBStore) Remove(alias string) error {
	items, err := s.readAll()
	if err != nil {
		return err
	}

	filtered := make([]DB, 0, len(items))
	removed := false
	for _, it := range items {
		if strings.EqualFold(it.Alias, alias) {
			removed = true
			continue
		}
		filtered = append(filtered, it)
	}
	if !removed {
		return NewValidationError(fmt.Sprintf("db alias %q was not found", alias))
	}

	return s.writeAll(filtered)
}

func (s *DBStore) Find(alias string) (DB, bool, error) {
	items, err := s.readAll()
	if err != nil {
		return DB{}, false, err
	}
	for _, it := range items {
		if strings.EqualFold(it.Alias, alias) {
			return it, true, nil
		}
	}
	return DB{}, false, nil
}

func (s *DBStore) readAll() ([]DB, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []DB{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []DB{}, nil
	}
	var items []DB
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *DBStore) writeAll(items []DB) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func validateBaseURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return NewValidationError("db url must be a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return NewValidationError("db url must start with http:// or https://")
	}
	if u.Host == "" {
		return NewValidationError("db url must include host")
	}
	return nil
}
