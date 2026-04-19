package storage

import (
	"fmt"
	"net/url"
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

func (s *DBStore) Update(currentAlias, nextAlias, baseURL string) error {
	if strings.TrimSpace(currentAlias) == "" {
		return NewValidationError("current db alias is required")
	}
	if strings.TrimSpace(nextAlias) == "" {
		return NewValidationError("db alias is required")
	}
	if err := validateBaseURL(baseURL); err != nil {
		return err
	}

	items, err := s.readAll()
	if err != nil {
		return err
	}

	target := -1
	for i, it := range items {
		if strings.EqualFold(it.Alias, currentAlias) {
			target = i
			continue
		}
		if strings.EqualFold(it.Alias, nextAlias) {
			return NewValidationError(fmt.Sprintf("db alias %q already exists", nextAlias))
		}
	}
	if target < 0 {
		return NewValidationError(fmt.Sprintf("db alias %q was not found", currentAlias))
	}

	items[target] = DB{Alias: nextAlias, BaseURL: baseURL}
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

func (s *DBStore) ReplaceAll(items []DB) error {
	cloned := make([]DB, len(items))
	copy(cloned, items)
	return s.writeAll(cloned)
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
	var items []DB
	found, err := readJSONFile(s.path, &items)
	if err != nil || !found {
		return []DB{}, err
	}
	return items, nil
}

func (s *DBStore) writeAll(items []DB) error {
	return writeJSONFile(s.path, items)
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
