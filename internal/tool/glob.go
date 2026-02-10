package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxGlobResults = 1000

// GlobTool finds files matching a glob pattern.
type GlobTool struct{}

type globParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

func (t *GlobTool) Name() string                     { return "glob" }
func (t *GlobTool) Description() string              { return "Find files matching a glob pattern" }
func (t *GlobTool) Permission() PermissionLevel      { return PermissionAuto }

func (t *GlobTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"pattern": {
			"type": "string",
			"description": "Glob pattern to match files (e.g., '**/*.go', 'src/*.ts')"
		},
		"path": {
			"type": "string",
			"description": "Directory to search in (default: current directory)"
		}
	},
	"required": ["pattern"]
}`)
}

func (t *GlobTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p globParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.Pattern == "" {
		return "Error: pattern is required", nil
	}

	dir := p.Path
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
	}

	var matches []string

	if strings.Contains(p.Pattern, "**") {
		// Recursive glob: split on ** and match suffix against walked files
		matches = recursiveGlob(dir, p.Pattern)
	} else {
		// Simple glob
		fullPattern := filepath.Join(dir, p.Pattern)
		var err error
		matches, err = filepath.Glob(fullPattern)
		if err != nil {
			return fmt.Sprintf("Error: invalid pattern: %v", err), nil
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No files matched the pattern: %s", p.Pattern), nil
	}

	truncated := false
	if len(matches) > maxGlobResults {
		matches = matches[:maxGlobResults]
		truncated = true
	}

	result := strings.Join(matches, "\n")
	if truncated {
		result += fmt.Sprintf("\n\n[truncated — showing first %d of more results]", maxGlobResults)
	}
	return result, nil
}

// recursiveGlob handles patterns containing **.
func recursiveGlob(root, pattern string) []string {
	// Split pattern on "**/" or "**"
	parts := strings.SplitN(pattern, "**", 2)
	prefix := parts[0]
	suffix := ""
	if len(parts) > 1 {
		suffix = parts[1]
		// Remove leading separator from suffix
		suffix = strings.TrimPrefix(suffix, "/")
		suffix = strings.TrimPrefix(suffix, string(filepath.Separator))
	}

	// If there's a prefix, adjust root
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		prefix = strings.TrimSuffix(prefix, string(filepath.Separator))
		root = filepath.Join(root, prefix)
	}

	var matches []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if len(matches) >= maxGlobResults*2 { // collect extra to handle sorting later
			return filepath.SkipAll
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != root {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if suffix == "" {
			matches = append(matches, path)
			return nil
		}

		// Match the suffix against the file name or relative path
		matched, _ := filepath.Match(suffix, d.Name())
		if matched {
			matches = append(matches, path)
			return nil
		}

		// Also try matching against relative path from root
		rel, err := filepath.Rel(root, path)
		if err == nil {
			matched, _ = filepath.Match(suffix, rel)
			if matched {
				matches = append(matches, path)
			}
		}

		return nil
	})
	return matches
}
