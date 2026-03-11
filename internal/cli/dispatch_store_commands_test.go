package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jiseop121/pbdash/internal/storage"
)

func TestDispatcherUpdateDBAliasCascadesSuperusersAndContext(t *testing.T) {
	dataDir := t.TempDir()
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: dataDir})

	_, err := d.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)
	_, err = d.saveSuperuser("dev", "root", "root@example.com", "pw")
	require.NoError(t, err)

	d.sessionCtx = commandContext{DBAlias: "dev", SuperuserAlias: "root"}
	require.NoError(t, d.persistSavedContext(d.sessionCtx))

	updated, err := d.updateDBAlias("dev", "prod", "https://pb.example.com")
	require.NoError(t, err)

	assert.Equal(t, "prod", updated.Alias)
	assert.Equal(t, "https://pb.example.com", updated.BaseURL)
	assert.Equal(t, "prod", d.sessionCtx.DBAlias)
	assert.Equal(t, "prod", d.savedCtx.DBAlias)

	renamedSuperuser, found, err := d.suStore.Find("prod", "root")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "prod", renamedSuperuser.DBAlias)
}

func TestDispatcherUpdateSuperuserKeepsPasswordWhenBlank(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})

	_, err := d.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)
	_, err = d.saveSuperuser("dev", "root", "root@example.com", "secret")
	require.NoError(t, err)

	d.sessionCtx = commandContext{DBAlias: "dev", SuperuserAlias: "root"}

	updated, err := d.updateSuperuser("dev", "root", "admin", "admin@example.com", "")
	require.NoError(t, err)

	assert.Equal(t, "admin", updated.Alias)
	assert.Equal(t, "admin@example.com", updated.Email)
	assert.Equal(t, "secret", updated.Password)
	assert.Equal(t, "admin", d.sessionCtx.SuperuserAlias)
}

func TestDispatcherRemoveDBAliasRemovesSuperusers(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})

	_, err := d.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)
	_, err = d.saveSuperuser("dev", "root", "root@example.com", "pw")
	require.NoError(t, err)

	require.NoError(t, d.removeDBAlias("dev"))

	_, foundDB, err := d.dbStore.Find("dev")
	require.NoError(t, err)
	assert.False(t, foundDB)

	items, err := d.suStore.ListByDB("dev")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestDispatcherUpdateDBAliasRollsBackOnSavedContextFailure(t *testing.T) {
	dataDir := t.TempDir()
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: dataDir})

	_, err := d.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)
	_, err = d.saveSuperuser("dev", "root", "root@example.com", "pw")
	require.NoError(t, err)

	d.sessionCtx = commandContext{DBAlias: "dev", SuperuserAlias: "root"}
	require.NoError(t, d.persistSavedContext(d.sessionCtx))

	saveCalls := 0
	d.saveSavedContext = func(ctx storage.Context) error {
		saveCalls++
		if saveCalls == 1 {
			return errors.New("boom")
		}
		return d.ctxStore.Save(ctx)
	}

	_, err = d.updateDBAlias("dev", "prod", "https://pb.example.com")
	require.Error(t, err)

	db, found, err := d.dbStore.Find("dev")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "http://127.0.0.1:8090", db.BaseURL)

	_, found, err = d.dbStore.Find("prod")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "dev", d.sessionCtx.DBAlias)
	assert.Equal(t, "dev", d.savedCtx.DBAlias)
}

func TestDispatcherUpdateSuperuserRollsBackOnSavedContextFailure(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})

	_, err := d.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)
	_, err = d.saveSuperuser("dev", "root", "root@example.com", "secret")
	require.NoError(t, err)

	d.sessionCtx = commandContext{DBAlias: "dev", SuperuserAlias: "root"}
	require.NoError(t, d.persistSavedContext(d.sessionCtx))

	saveCalls := 0
	d.saveSavedContext = func(ctx storage.Context) error {
		saveCalls++
		if saveCalls == 1 {
			return errors.New("boom")
		}
		return d.ctxStore.Save(ctx)
	}

	_, err = d.updateSuperuser("dev", "root", "admin", "admin@example.com", "")
	require.Error(t, err)

	su, found, err := d.suStore.Find("dev", "root")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "root@example.com", su.Email)
	assert.Equal(t, "root", d.sessionCtx.SuperuserAlias)
	assert.Equal(t, "root", d.savedCtx.SuperuserAlias)
}

func TestDispatcherRemoveDBAliasRollsBackOnSavedContextClearFailure(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{Stdout: bytes.NewBuffer(nil), Version: "test", DataDir: t.TempDir()})

	_, err := d.saveDBAlias("dev", "http://127.0.0.1:8090")
	require.NoError(t, err)
	_, err = d.saveSuperuser("dev", "root", "root@example.com", "pw")
	require.NoError(t, err)

	d.sessionCtx = commandContext{DBAlias: "dev", SuperuserAlias: "root"}
	require.NoError(t, d.persistSavedContext(d.sessionCtx))

	clearCalls := 0
	d.clearSavedContext = func() error {
		clearCalls++
		if clearCalls == 1 {
			return errors.New("boom")
		}
		return d.ctxStore.Clear()
	}

	err = d.removeDBAlias("dev")
	require.Error(t, err)

	_, found, err := d.dbStore.Find("dev")
	require.NoError(t, err)
	assert.True(t, found)

	items, err := d.suStore.ListByDB("dev")
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.True(t, d.hasSaved)
	assert.Equal(t, "dev", d.savedCtx.DBAlias)
}
