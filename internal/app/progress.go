package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// progressFileName is the file where the hiragana learning progress (the set
// of kana the user already knows) is persisted between application runs. It
// lives in the user's OS configuration directory and is git-ignored by name.
const progressFileName = "japan_progress.json"

// ProgressData is the on-disk representation of the learning state.
type ProgressData struct {
	KnownHiragana []string `json:"known_hiragana"`
}

// progressDir returns the directory used to store the progress file. Tests
// override os.UserConfigDir behavior by pointing HOME/XDG variables at a
// temporary directory, so the real implementation stays simple here.
func progressDir() (string, error) {
	return os.UserConfigDir()
}

// ProgressPath returns the full path of the progress file.
func ProgressPath() (string, error) {
	dir, err := progressDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "japan", progressFileName), nil
}

// LoadProgress reads the persisted progress. A missing or unreadable/corrupt
// file is treated as "no progress yet" so the app always starts cleanly.
func LoadProgress() ProgressData {
	path, err := ProgressPath()
	if err != nil {
		return ProgressData{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ProgressData{}
	}
	var p ProgressData
	if err := json.Unmarshal(raw, &p); err != nil {
		return ProgressData{}
	}
	return p
}

// SaveProgress atomically-ish writes the progress to disk, creating the
// containing directory if needed. Errors are returned to the caller which may
// choose to ignore them (losing progress must not crash the quiz).
func SaveProgress(p ProgressData) error {
	path, err := ProgressPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// ResetProgress deletes the persisted progress file. A missing file is not an
// error: after a reset there simply is no progress.
func ResetProgress() error {
	path, err := ProgressPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
