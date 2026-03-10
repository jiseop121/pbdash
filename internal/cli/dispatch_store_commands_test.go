package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
