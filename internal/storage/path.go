package storage

import (
	"os"
	"path/filepath"
)

// ResolveAppDir returns a writable directory for the given subdir.
// Priority: executable sibling -> user config/home -> relative dir.
func ResolveAppDir(subdir string) string {
	// Executable sibling
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		dir := filepath.Join(exeDir, subdir)
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}

	// User config dir
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(userConfigDir, "iOSGhostRun", subdir)
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}

	// Home dir fallback
	if homeDir, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(homeDir, ".iosghostrun", subdir)
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}

	// Relative fallback
	dir := filepath.Join(subdir)
	_ = os.MkdirAll(dir, 0755)
	return dir
}
