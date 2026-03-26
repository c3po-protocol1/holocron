package claudecode

import (
	"os"
	"path/filepath"
	"strings"
)

// pathExistsFunc is the default filesystem existence check, overridable for testing.
var pathExistsFunc = func(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

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
// Claude Code slugs are formed by replacing both "/" and "." with "-".
// Example: "-Users-c-3po-Projects-holocron" → "/Users/c-3po/Projects/holocron"
// Since "-" is ambiguous (could be literal, "/" or "."), we resolve against the filesystem.
func SlugToPath(slug string) string {
	return slugToPath(slug, pathExistsFunc)
}

func slugToPath(slug string, exists func(string) bool) string {
	if slug == "" {
		return ""
	}
	if !strings.HasPrefix(slug, "-") {
		return slug
	}
	return resolveSlugPath("/", slug[1:], exists)
}

// resolveSlugPath reconstructs a filesystem path from slug segments by greedily
// matching the longest existing path component at each level.
func resolveSlugPath(base, remaining string, exists func(string) bool) string {
	if remaining == "" {
		return base
	}

	// Collect "-" positions (potential component boundaries), longest first.
	splits := []int{len(remaining)}
	for i := len(remaining) - 1; i >= 0; i-- {
		if remaining[i] == '-' {
			splits = append(splits, i)
		}
	}

	for _, pos := range splits {
		component := remaining[:pos]
		if component == "" {
			continue
		}

		var rest string
		if pos < len(remaining) {
			rest = remaining[pos+1:]
		}

		// Try component as-is (literal hyphens preserved).
		candidate := filepath.Join(base, component)
		if exists(candidate) {
			result := resolveSlugPath(candidate, rest, exists)
			if exists(result) || rest == "" {
				return result
			}
		}

		// Try interpreting a leading "-" as "." (hidden dir/file).
		if strings.HasPrefix(component, "-") {
			dotCandidate := filepath.Join(base, "."+component[1:])
			if exists(dotCandidate) {
				result := resolveSlugPath(dotCandidate, rest, exists)
				if exists(result) || rest == "" {
					return result
				}
			}
		}
	}

	// Fallback: filesystem resolution failed; replace remaining "-" with "/".
	fallback := base + "/" + strings.ReplaceAll(remaining, "-", "/")
	return filepath.Clean(fallback)
}
