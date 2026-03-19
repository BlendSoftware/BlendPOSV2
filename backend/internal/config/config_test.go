package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_ReadReplicaURL_Parsed verifies that DATABASE_READ_REPLICA_URL is
// correctly loaded into the Config struct when set as an env var.
func TestLoad_ReadReplicaURL_Parsed(t *testing.T) {
	const replicaURL = "postgres://reader:secret@replica-host:5432/blendpos?sslmode=require"

	// Set required env vars for Load() to succeed
	t.Setenv("JWT_SECRET", "test-secret-key-must-be-at-least-32-chars-long!!")
	t.Setenv("DATABASE_READ_REPLICA_URL", replicaURL)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, replicaURL, cfg.DatabaseReadReplicaURL)
}

// TestLoad_ReadReplicaURL_EmptyByDefault verifies that when
// DATABASE_READ_REPLICA_URL is not set, the field is empty (callers use
// fallback logic).
func TestLoad_ReadReplicaURL_EmptyByDefault(t *testing.T) {
	// Set required env vars for Load() to succeed
	t.Setenv("JWT_SECRET", "test-secret-key-must-be-at-least-32-chars-long!!")

	// Explicitly unset so a previous test or env doesn't leak
	os.Unsetenv("DATABASE_READ_REPLICA_URL")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.DatabaseReadReplicaURL, "should be empty when env var is not set")
}
