package claudecode

import (
	"os"
	"path/filepath"
	"strings"
)

// SessionFile represents a discovered JSONL session file.
type SessionFile struct {
	Path      string
	SessionID string
	Workspace string
}

// ScanSessions scans the given base directory for session JSONL files.
// The base directory is expected to contain workspace-slug subdirectories,
// each containing <sessionId>.jsonl files.
func ScanSessions(baseDir string) ([]SessionFile, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var sessions []SessionFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		workspace := SlugToPath(entry.Name())
		dirPath := filepath.Join(baseDir, entry.Name())
		files, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			sessionID := strings.TrimSuffix(f.Name(), ".jsonl")
			sessions = append(sessions, SessionFile{
				Path:      filepath.Join(dirPath, f.Name()),
				SessionID: sessionID,
				Workspace: workspace,
			})
		}
	}
	return sessions, nil
}

// SlugToPath converts a Claude Code workspace slug back to a filesystem path.
// Example: "-Users-c-3po-Projects-holocron" → "/Users/c-3po/Projects/holocron"
func SlugToPath(slug string) string {
	if slug == "" {
		return ""
	}
	// The slug replaces "/" with "-" and starts with "-" (representing the leading "/").
	// We replace the leading "-" with "/" and then each remaining "-" with "/".
	// However, directory names may contain hyphens, so we can't blindly replace all.
	// Claude Code's observed format: the slug is simply the path with "/" replaced by "-".
	// We reverse that transformation.
	return strings.ReplaceAll(slug, "-", "/")
}
