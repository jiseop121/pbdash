package storage

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuperuserStoreEncryptsPasswordAtRest(t *testing.T) {
	dir := t.TempDir()
	store := NewSuperuserStore(dir)

	if err := store.Add("dev", "root", "root@example.com", "secret-password"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "superusers.json"))
	if err != nil {
		t.Fatalf("read superusers.json: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "secret-password") {
		t.Fatalf("password should not be stored in plaintext: %s", text)
	}
	if strings.Contains(text, `"password"`) {
		t.Fatalf("legacy password field should not be persisted: %s", text)
	}
	if !strings.Contains(text, `"passwordEnc"`) {
		t.Fatalf("encrypted password field missing: %s", text)
	}

	found, ok, err := store.Find("dev", "root")
	if err != nil {
		t.Fatalf("find superuser: %v", err)
	}
	if !ok {
		t.Fatalf("expected superuser to be found")
	}
	if found.Password != "secret-password" {
		t.Fatalf("decrypted password mismatch: got=%q", found.Password)
	}
}

func TestSuperuserStoreMigratesLegacyPlaintextOnWrite(t *testing.T) {
	dir := t.TempDir()
	legacy := `[
  {
    "dbAlias": "dev",
    "alias": "legacy",
    "email": "legacy@example.com",
    "password": "legacy-password"
  }
]
`
	if err := os.WriteFile(filepath.Join(dir, "superusers.json"), []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy superusers.json: %v", err)
	}

	store := NewSuperuserStore(dir)
	if err := store.Add("dev", "new", "new@example.com", "new-password"); err != nil {
		t.Fatalf("add superuser with legacy file: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "superusers.json"))
	if err != nil {
		t.Fatalf("read superusers.json: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "legacy-password") || strings.Contains(text, "new-password") {
		t.Fatalf("plaintext password should be migrated out: %s", text)
	}
	if strings.Contains(text, `"password"`) {
		t.Fatalf("legacy password field should be removed after write: %s", text)
	}
}

func TestSuperuserStoreUsesEnvKeyWithoutPersistingKeyFile(t *testing.T) {
	key := strings.Repeat("k", 32)
	encoded := base64.StdEncoding.EncodeToString([]byte(key))
	t.Setenv("PBDASH_SUPERUSER_KEY_B64", encoded)

	dir := t.TempDir()
	store := NewSuperuserStore(dir)
	if err := store.Add("dev", "root", "root@example.com", "pw123456"); err != nil {
		t.Fatalf("add superuser: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "superusers.key")); !os.IsNotExist(err) {
		t.Fatalf("key file should not be created when env key is set: %v", err)
	}

	got, ok, err := store.Find("dev", "root")
	if err != nil {
		t.Fatalf("find superuser: %v", err)
	}
	if !ok {
		t.Fatalf("expected superuser to be found")
	}
	if got.Password != "pw123456" {
		t.Fatalf("decrypted password mismatch: got=%q", got.Password)
	}
}

func TestSuperuserStoreRejectsInvalidEnvKey(t *testing.T) {
	t.Setenv("PBDASH_SUPERUSER_KEY_B64", "invalid")
	store := NewSuperuserStore(t.TempDir())
	err := store.Add("dev", "root", "root@example.com", "pw123456")
	if err == nil {
		t.Fatalf("expected error for invalid env key")
	}
	if !strings.Contains(err.Error(), "PBDASH_SUPERUSER_KEY_B64") {
		t.Fatalf("missing env key context: %v", err)
	}
}

func TestSuperuserStoreUpdateKeepsExistingPasswordWhenBlank(t *testing.T) {
	store := NewSuperuserStore(t.TempDir())
	require.NoError(t, store.Add("dev", "root", "root@example.com", "secret"))

	require.NoError(t, store.Update("dev", "root", "admin", "admin@example.com", ""))

	updated, found, err := store.Find("dev", "admin")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "admin@example.com", updated.Email)
	assert.Equal(t, "secret", updated.Password)
}

func TestSuperuserStoreReassignDBAlias(t *testing.T) {
	store := NewSuperuserStore(t.TempDir())
	require.NoError(t, store.Add("dev", "root", "root@example.com", "secret"))

	require.NoError(t, store.ReassignDBAlias("dev", "prod"))

	updated, found, err := store.Find("prod", "root")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "prod", updated.DBAlias)
}
