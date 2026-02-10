package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxGrepMatches = 500

// skipDirs contains directories to skip during grep traversal.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".hg":          true,
	".svn":         true,
	"__pycache__":  true,
}

// GrepTool searches file contents with regex.
type GrepTool struct{}

type grepParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Include string `json:"include"`
}

func (t *GrepTool) Name() string                { return "grep" }
func (t *GrepTool) Description() string         { return "Search file contents using a regex pattern" }
func (t *GrepTool) Permission() PermissionLevel { return PermissionAuto }

func (t *GrepTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"pattern": {
			"type": "string",
			"description": "Regex pattern to search for"
		},
		"path": {
			"type": "string",
			"description": "File or directory to search in (default: current directory)"
		},
		"include": {
			"type": "string",
			"description": "Glob pattern to filter files (e.g., '*.go')"
		}
	},
	"required": ["pattern"]
}`)
}

func (t *GrepTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p grepParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.Pattern == "" {
		return "Error: pattern is required", nil
	}

	re, err := regexp.Compile(p.Pattern)
	if err != nil {
		return fmt.Sprintf("Error: invalid regex: %v", err), nil
	}

	searchPath := p.Path
	if searchPath == "" {
		searchPath, err = os.Getwd()
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
	}

	info, err := os.Stat(searchPath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	var matches []string
	if info.IsDir() {
		matches = grepDir(searchPath, re, p.Include)
	} else {
		matches = grepFile(searchPath, re)
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for pattern: %s", p.Pattern), nil
	}

	truncated := false
	if len(matches) > maxGrepMatches {
		matches = matches[:maxGrepMatches]
		truncated = true
	}

	result := strings.Join(matches, "\n")
	if truncated {
		result += fmt.Sprintf("\n\n[truncated — showing first %d matches]", maxGrepMatches)
	}
	return result, nil
}

func grepDir(dir string, re *regexp.Regexp, include string) []string {
	var matches []string

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if len(matches) >= maxGrepMatches*2 {
			return filepath.SkipAll
		}

		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] || (strings.HasPrefix(name, ".") && path != dir) {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply include filter
		if include != "" {
			matched, _ := filepath.Match(include, d.Name())
			if !matched {
				return nil
			}
		}

		// Skip binary files
		if isBinary(path) {
			return nil
		}

		fileMatches := grepFile(path, re)
		matches = append(matches, fileMatches...)
		return nil
	})
	return matches
}

func grepFile(path string, re *regexp.Regexp) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var matches []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%s:%d:%s", path, lineNum, line))
		}
	}
	return matches
}

// isBinary checks if a file appears to be binary by looking for null bytes
// in the first 512 bytes.
func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false
	}
	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}
