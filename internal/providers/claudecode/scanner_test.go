package claudecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlugToPath(t *testing.T) {
	// Mock filesystem for deterministic tests.
	existing := map[string]bool{
		"/Users":                              true,
		"/Users/c-3po":                        true,
		"/Users/c-3po/Projects":               true,
		"/Users/c-3po/Projects/holocron":      true,
		"/Users/c-3po/Projects/other":         true,
		"/Users/c-3po/.openclaw":              true,
		"/Users/c-3po/.openclaw/workspace":    true,
	}
	exists := func(path string) bool { return existing[path] }

	tests := []struct {
		name string
		slug string
		want string
	}{
		{
			name: "path with hyphens in dir name",
			slug: "-Users-c-3po-Projects-holocron",
			want: "/Users/c-3po/Projects/holocron",
		},
		{
			name: "hidden directory (dot-prefixed)",
			slug: "-Users-c-3po--openclaw-workspace",
			want: "/Users/c-3po/.openclaw/workspace",
		},
		{
			name: "empty slug",
			slug: "",
			want: "",
		},
		{
			name: "path without ambiguous hyphens",
			slug: "-Users-c-3po-Projects-other",
			want: "/Users/c-3po/Projects/other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugToPath(tt.slug, exists)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlugToPath_FallbackWhenNoFilesystem(t *testing.T) {
	// When nothing exists on disk, falls back to simple replacement.
	nothingExists := func(string) bool { return false }
	got := slugToPath("-foo-bar-baz", nothingExists)
	assert.Equal(t, "/foo/bar/baz", got)
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
	assert.NotEmpty(t, abc.Workspace)
	assert.Contains(t, abc.Path, "session-abc.jsonl")

	xyz := sessionMap["session-xyz"]
	assert.NotEmpty(t, xyz.Workspace)
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
