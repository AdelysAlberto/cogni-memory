package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cleanProjectName(name string) string {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" || cleaned == "/" || cleaned == "." || cleaned == "\\" {
		return ""
	}
	return cleaned
}

// DetectProjectName inspects the workspace to determine the current project identifier
func DetectProjectName() string {
	// 1. Try git root directory name
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		gitRoot := strings.TrimSpace(string(out))
		if cleaned := cleanProjectName(filepath.Base(gitRoot)); cleaned != "" {
			return cleaned
		}
	}

	// 2. Try package.json
	if data, err := os.ReadFile("package.json"); err == nil {
		var pkg struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &pkg); err == nil {
			if cleaned := cleanProjectName(pkg.Name); cleaned != "" {
				return cleaned
			}
		}
	}

	// 3. Try go.mod
	if data, err := os.ReadFile("go.mod"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "module ") {
				modName := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
				parts := strings.Split(modName, "/")
				if cleaned := cleanProjectName(parts[len(parts)-1]); cleaned != "" {
					return cleaned
				}
			}
		}
	}

	// 4. Fallback to current working directory name
	if cwd, err := os.Getwd(); err == nil {
		if cleaned := cleanProjectName(filepath.Base(cwd)); cleaned != "" {
			return cleaned
		}
	}

	return "default_project"
}

// FindLocalCogniDir searches for .cogni in current directory or parent directories up to root/git root
func FindLocalCogniDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(dir, ".cogni")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// GetGlobalCogniDir returns the ~/.cogni directory path
func GetGlobalCogniDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cogni"
	}
	return filepath.Join(home, ".cogni")
}

// ResolveDatabasePath returns the appropriate SQLite database path
func ResolveDatabasePath(customPath string, forceGlobal bool) string {
	if customPath != "" {
		return customPath
	}

	if forceGlobal {
		globalDir := GetGlobalCogniDir()
		_ = os.MkdirAll(globalDir, 0755)
		return filepath.Join(globalDir, "memory.db")
	}

	// Check if local .cogni exists
	localDir := FindLocalCogniDir()
	if localDir != "" {
		return filepath.Join(localDir, "memory.db")
	}

	// Default to global
	globalDir := GetGlobalCogniDir()
	_ = os.MkdirAll(globalDir, 0755)
	return filepath.Join(globalDir, "memory.db")
}

// FormatTags ensures tags are normalized and includes project tag if missing
func FormatTags(tags string, projectName string) string {
	rawParts := strings.Split(tags, ",")
	seen := make(map[string]bool)
	var cleanParts []string

	// Add project tag if valid
	projectTag := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(projectName), " ", "-"))
	if projectTag != "" && projectTag != "global" && projectTag != "default_project" && projectTag != "/" && projectTag != "." {
		seen[projectTag] = true
		cleanParts = append(cleanParts, projectTag)
	}

	for _, p := range rawParts {
		tag := strings.ToLower(strings.TrimSpace(p))
		tag = strings.TrimPrefix(tag, "#")
		tag = strings.ReplaceAll(tag, " ", "-")
		if tag != "" && !seen[tag] {
			seen[tag] = true
			cleanParts = append(cleanParts, tag)
		}
	}

	return strings.Join(cleanParts, ",")
}

// EstimateTokens calculates estimated saved tokens based on content length
func EstimateTokens(text string) int64 {
	// Average rule: ~4 chars per token
	return int64(len([]rune(text)) / 4)
}
