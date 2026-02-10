// Package memory handles loading persistent memory from
// .stormtrooper/memory/ for injection into the system prompt.
package memory

import (
	"os"
	"path/filepath"
)

const memoryDir = ".stormtrooper/memory"
const memoryFile = "MEMORY.md"

// Load reads the MEMORY.md file from .stormtrooper/memory/ in the given
// project directory. Returns the content as a string, or empty string if
// the file doesn't exist.
func Load(projectDir string) (string, error) {
	path := filepath.Join(projectDir, memoryDir, memoryFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Dir returns the absolute path to the memory directory for the given
// project directory.
func Dir(projectDir string) string {
	return filepath.Join(projectDir, memoryDir)
}
