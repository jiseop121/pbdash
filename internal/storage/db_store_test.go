package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBStoreAddFindListRemove(t *testing.T) {
	store := NewDBStore(t.TempDir())

	if err := store.Add("dev", "http://127.0.0.1:8090"); err != nil {
		t.Fatalf("add dev: %v", err)
	}
	if err := store.Add("Prod", "https://example.com"); err != nil {
		t.Fatalf("add Prod: %v", err)
	}

	found, ok, err := store.Find("DEV")
	if err != nil {
		t.Fatalf("find dev: %v", err)
	}
	if !ok {
		t.Fatalf("expected dev to exist")
	}
	if found.BaseURL != "http://127.0.0.1:8090" {
		t.Fatalf("unexpected base url: %q", found.BaseURL)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("list length mismatch: got=%d want=2", len(items))
	}
	if items[0].Alias != "dev" || items[1].Alias != "Prod" {
		t.Fatalf("unexpected alias order: %+v", items)
	}

	if err := store.Remove("pRoD"); err != nil {
		t.Fatalf("remove Prod: %v", err)
	}
	if _, ok, err := store.Find("prod"); err != nil {
		t.Fatalf("find prod after remove: %v", err)
	} else if ok {
		t.Fatalf("expected prod to be removed")
	}
}

func TestDBStoreValidation(t *testing.T) {
	store := NewDBStore(t.TempDir())

	if err := store.Add("", "http://127.0.0.1:8090"); err == nil {
		t.Fatalf("expected alias validation error")
	}
	if err := store.Add("dev", "127.0.0.1:8090"); err == nil {
		t.Fatalf("expected url validation error")
	}
	if err := store.Add("dev", "http://127.0.0.1:8090"); err != nil {
		t.Fatalf("add dev: %v", err)
	}
	if err := store.Add("DEV", "http://127.0.0.1:8091"); err == nil {
		t.Fatalf("expected duplicate alias validation error")
	}
	if err := store.Remove("missing"); err == nil {
		t.Fatalf("expected remove missing validation error")
	}
}

func TestDBStoreUpdate(t *testing.T) {
	store := NewDBStore(t.TempDir())
	require.NoError(t, store.Add("dev", "http://127.0.0.1:8090"))

	require.NoError(t, store.Update("dev", "prod", "https://pb.example.com"))

	updated, found, err := store.Find("prod")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "https://pb.example.com", updated.BaseURL)
}
