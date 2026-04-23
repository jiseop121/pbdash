package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Superuser struct {
	DBAlias     string `json:"dbAlias"`
	Alias       string `json:"alias"`
	Email       string `json:"email"`
	Password    string `json:"password,omitempty"`
	PasswordEnc string `json:"passwordEnc,omitempty"`
}

type SuperuserStore struct {
	path    string
	keyPath string
}

const superuserKeyEnv = "PBDASH_SUPERUSER_KEY_B64"

func NewSuperuserStore(dataDir string) *SuperuserStore {
	return &SuperuserStore{
		path:    filepath.Join(dataDir, "superusers.json"),
		keyPath: filepath.Join(dataDir, "superusers.key"),
	}
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

	key, err := s.loadOrCreateKey()
	if err != nil {
		return err
	}
	encrypted, err := encryptPassword(key, password)
	if err != nil {
		return err
	}

	items = append(items, Superuser{
		DBAlias:     dbAlias,
		Alias:       alias,
		Email:       email,
		Password:    password,
		PasswordEnc: encrypted,
	})
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

func (s *SuperuserStore) List() ([]Superuser, error) {
	items, err := s.readAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		left := strings.ToLower(items[i].DBAlias + "/" + items[i].Alias)
		right := strings.ToLower(items[j].DBAlias + "/" + items[j].Alias)
		return left < right
	})
	return items, nil
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

func (s *SuperuserStore) Update(dbAlias, currentAlias, nextAlias, email, password string) error {
	if strings.TrimSpace(dbAlias) == "" || strings.TrimSpace(currentAlias) == "" || strings.TrimSpace(nextAlias) == "" || strings.TrimSpace(email) == "" {
		return NewValidationError("--db, current alias, alias, and --email are required")
	}

	items, err := s.readAll()
	if err != nil {
		return err
	}

	target := -1
	for i, it := range items {
		if strings.EqualFold(it.DBAlias, dbAlias) && strings.EqualFold(it.Alias, currentAlias) {
			target = i
			continue
		}
		if strings.EqualFold(it.DBAlias, dbAlias) && strings.EqualFold(it.Alias, nextAlias) {
			return NewValidationError(fmt.Sprintf("superuser alias %q already exists for db %q", nextAlias, dbAlias))
		}
	}
	if target < 0 {
		return NewValidationError(fmt.Sprintf("superuser alias %q is not configured for db %q", currentAlias, dbAlias))
	}

	// 비밀번호 미입력 시 기존 암호화값을 그대로 사용 (복호화된 평문을 불필요하게 보유하지 않음)
	if strings.TrimSpace(password) == "" {
		items[target] = Superuser{
			DBAlias:     dbAlias,
			Alias:       nextAlias,
			Email:       email,
			PasswordEnc: items[target].PasswordEnc,
		}
	} else {
		items[target] = Superuser{
			DBAlias:  dbAlias,
			Alias:    nextAlias,
			Email:    email,
			Password: password,
		}
	}
	return s.writeAll(items)
}

func (s *SuperuserStore) RemoveByDB(dbAlias string) error {
	items, err := s.readAll()
	if err != nil {
		return err
	}

	filtered := make([]Superuser, 0, len(items))
	removed := false
	for _, it := range items {
		if strings.EqualFold(it.DBAlias, dbAlias) {
			removed = true
			continue
		}
		filtered = append(filtered, it)
	}
	if !removed {
		return nil
	}
	return s.writeAll(filtered)
}

func (s *SuperuserStore) ReassignDBAlias(currentAlias, nextAlias string) error {
	if strings.TrimSpace(currentAlias) == "" || strings.TrimSpace(nextAlias) == "" {
		return NewValidationError("current db alias and next db alias are required")
	}

	items, err := s.readAll()
	if err != nil {
		return err
	}

	seen := map[string]struct{}{}
	updated := false
	for _, it := range items {
		dbAlias := it.DBAlias
		if strings.EqualFold(dbAlias, currentAlias) {
			dbAlias = nextAlias
			updated = true
		}
		key := strings.ToLower(strings.TrimSpace(dbAlias) + "\x00" + strings.TrimSpace(it.Alias))
		if _, ok := seen[key]; ok {
			return NewValidationError(fmt.Sprintf("superuser alias %q already exists for db %q", it.Alias, nextAlias))
		}
		seen[key] = struct{}{}
	}
	if !updated {
		return nil
	}

	for i := range items {
		if strings.EqualFold(items[i].DBAlias, currentAlias) {
			items[i].DBAlias = nextAlias
		}
	}
	return s.writeAll(items)
}

func (s *SuperuserStore) ReplaceAll(items []Superuser) error {
	cloned := make([]Superuser, len(items))
	copy(cloned, items)
	return s.writeAll(cloned)
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
	var items []Superuser
	found, err := readJSONFile(s.path, &items)
	if err != nil {
		return nil, err
	}
	if !found {
		return []Superuser{}, nil
	}

	needsDecryption := false
	for i := range items {
		if strings.TrimSpace(items[i].PasswordEnc) != "" {
			needsDecryption = true
			break
		}
	}

	var key []byte
	if needsDecryption {
		key, err = s.loadOrCreateKey()
		if err != nil {
			return nil, err
		}
	}

	for i := range items {
		switch {
		case strings.TrimSpace(items[i].PasswordEnc) != "":
			password, decErr := decryptPassword(key, items[i].PasswordEnc)
			if decErr != nil {
				return nil, fmt.Errorf("decrypt superuser password for %q/%q: %w", items[i].DBAlias, items[i].Alias, decErr)
			}
			items[i].Password = password
		case strings.TrimSpace(items[i].Password) != "":
			// Legacy plaintext format. Keep compatibility and re-encrypt on next write.
		default:
			return nil, fmt.Errorf("superuser %q for db %q has no usable credential", items[i].Alias, items[i].DBAlias)
		}
	}
	return items, nil
}

func (s *SuperuserStore) writeAll(items []Superuser) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	key, err := s.loadOrCreateKey()
	if err != nil {
		return err
	}
	persisted := make([]Superuser, 0, len(items))
	for _, item := range items {
		plain := strings.TrimSpace(item.Password)
		cipherText := strings.TrimSpace(item.PasswordEnc)
		if plain == "" && cipherText == "" {
			return fmt.Errorf("superuser %q for db %q has no usable credential", item.Alias, item.DBAlias)
		}
		if plain != "" {
			cipherText, err = encryptPassword(key, plain)
			if err != nil {
				return err
			}
		}
		persisted = append(persisted, Superuser{
			DBAlias:     item.DBAlias,
			Alias:       item.Alias,
			Email:       item.Email,
			PasswordEnc: cipherText,
		})
	}

	return writeJSONFile(s.path, persisted)
}

func (s *SuperuserStore) loadOrCreateKey() ([]byte, error) {
	if envKey := strings.TrimSpace(os.Getenv(superuserKeyEnv)); envKey != "" {
		key, err := decodeStoredKey(envKey)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", superuserKeyEnv, err)
		}
		return key, nil
	}

	if data, err := os.ReadFile(s.keyPath); err == nil {
		key, decErr := decodeStoredKey(string(data))
		if decErr != nil {
			return nil, fmt.Errorf("invalid superuser key file: %w", decErr)
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(s.keyPath), 0o700); err != nil {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(key) + "\n"
	if err := os.WriteFile(s.keyPath, []byte(encoded), 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func decodeStoredKey(raw string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: %d", len(key))
	}
	return key, nil
}

func encryptPassword(key []byte, plain string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	blob := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(blob), nil
}

func decryptPassword(key []byte, encoded string) (string, error) {
	blob, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(blob) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
