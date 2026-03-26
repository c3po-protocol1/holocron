package claudecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlugToPath(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"-Users-c-3po-Projects-holocron", "/Users/c/3po/Projects/holocron"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := SlugToPath(tt.slug)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScanSessions(t *testing.T) {
	// Create mock directory structure
	baseDir := t.TempDir()

	// Create workspace directories with session files
	ws1 := filepath.Join(baseDir, "-Users-c-3po-Projects-holocron")
	require.NoError(t, os.MkdirAll(ws1, 0o755))

	ws2 := filepath.Join(baseDir, "-Users-c-3po-Projects-other")
	require.NoError(t, os.MkdirAll(ws2, 0o755))

	// Create session files
	require.NoError(t, os.WriteFile(filepath.Join(ws1, "session-abc.jsonl"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ws1, "session-def.jsonl"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ws2, "session-xyz.jsonl"), []byte(`{}`), 0o644))

	// Create a non-jsonl file that should be ignored
	require.NoError(t, os.WriteFile(filepath.Join(ws1, "notes.txt"), []byte("ignore"), 0o644))

	sessions, err := ScanSessions(baseDir)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Verify session details
	sessionMap := make(map[string]SessionFile)
	for _, s := range sessions {
		sessionMap[s.SessionID] = s
	}

	abc := sessionMap["session-abc"]
	assert.Equal(t, "/Users/c/3po/Projects/holocron", abc.Workspace)
	assert.Contains(t, abc.Path, "session-abc.jsonl")

	xyz := sessionMap["session-xyz"]
	assert.Equal(t, "/Users/c/3po/Projects/other", xyz.Workspace)
}

func TestScanSessions_EmptyDir(t *testing.T) {
	baseDir := t.TempDir()
	sessions, err := ScanSessions(baseDir)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestScanSessions_NonExistentDir(t *testing.T) {
	_, err := ScanSessions("/nonexistent/path")
	assert.Error(t, err)
}
