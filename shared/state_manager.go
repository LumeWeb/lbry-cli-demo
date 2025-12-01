package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// StateManager handles state persistence and retrieval
type StateManager struct {
	stateDir string
	logger   *zap.Logger
	mu       sync.RWMutex
}

// Constants for state file names
const (
	AccountStateFile = "account.json"
)

// AccountState holds account information for reuse
type AccountState struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	CreatedAt string `json:"created_at"`
	JWT       string `json:"jwt_token"`
}

// NewStateManager creates a new state manager
func NewStateManager(logger *zap.Logger) (*StateManager, error) {
	// Ensure marker file exists for future runs
	err := ensureMarkerFile()
	if err != nil {
		logger.Warn("Failed to ensure marker file", zap.Error(err))
	}

	// Determine state directory location
	stateDir, err := getStateDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine state directory: %w", err)
	}

	// Create state directory if it doesn't exist
	err = os.MkdirAll(stateDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	logger.Debug("State manager initialized", zap.String("state_dir", stateDir))

	return &StateManager{
		stateDir: stateDir,
		logger:   logger,
	}, nil
}

// ensureMarkerFile creates the marker file if it doesn't exist
func ensureMarkerFile() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Try to find existing marker file
	_, err = findMarkerDirectory(cwd, ".lbry-demo")
	if err == nil {
		// Marker file already exists
		return nil
	}

	// Create marker file in current directory
	markerPath := filepath.Join(cwd, ".lbry-demo")
	err = os.WriteFile(markerPath, []byte("# LBRY Demo Project Marker\n"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	return nil
}

// getStateDirectory determines the appropriate state directory location
func getStateDirectory() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Look for .lbry-demo marker file to find shared root
	markerDir, err := findMarkerDirectory(cwd, ".lbry-demo")
	if err != nil {
		// Fallback to current working directory
		return filepath.Join(cwd, "state"), nil
	}

	return filepath.Join(markerDir, "state"), nil
}

// findMarkerDirectory searches for a marker file to find the shared root
func findMarkerDirectory(startDir, markerFile string) (string, error) {
	dir := startDir
	for {
		// Check if marker file exists in this directory
		markerPath := filepath.Join(dir, markerFile)
		if _, err := os.Stat(markerPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root
			break
		}
		dir = parent

		// Prevent infinite loop by checking reasonable depth
		if len(filepath.SplitList(dir)) > 10 {
			break
		}
	}

	return "", fmt.Errorf("marker directory not found (no %s found)", markerFile)
}

// GetStatePath returns the full path for a state file
func (sm *StateManager) GetStatePath(filename string) string {
	return filepath.Join(sm.stateDir, filename)
}

// GetStateDir returns the state directory path
func (sm *StateManager) GetStateDir() string {
	return sm.stateDir
}

// SaveAccountState saves account information to account.json
func (sm *StateManager) SaveAccountState(account *FakeAccountData) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	accountState := &AccountState{
		Email:     account.Email,
		Password:  account.Password,
		FirstName: account.FirstName,
		LastName:  account.LastName,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	accountPath := sm.GetStatePath(AccountStateFile)
	data, err := json.MarshalIndent(accountState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account state: %w", err)
	}

	err = os.WriteFile(accountPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write account state file: %w", err)
	}

	sm.logger.Info("Account state saved", zap.String("path", accountPath))
	return nil
}

// LoadAccountState loads account information from account.json
func (sm *StateManager) LoadAccountState() (*AccountState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	accountPath := sm.GetStatePath(AccountStateFile)
	data, err := os.ReadFile(accountPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No account state exists
		}
		return nil, fmt.Errorf("failed to read account state file: %w", err)
	}

	var accountState AccountState
	err = json.Unmarshal(data, &accountState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account state: %w", err)
	}

	sm.logger.Info("Account state loaded", zap.String("path", accountPath))
	return &accountState, nil
}

// SaveJSON saves any JSON-serializable data to a state file
func (sm *StateManager) SaveJSON(filename string, data interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data for %s: %w", filename, err)
	}

	path := sm.GetStatePath(filename)
	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file %s: %w", filename, err)
	}

	sm.logger.Debug("State saved", zap.String("file", filename), zap.String("path", path))
	return nil
}

// LoadJSON loads JSON data from a state file
func (sm *StateManager) LoadJSON(filename string, target interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	path := sm.GetStatePath(filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, return nil
		}
		return fmt.Errorf("failed to read state file %s: %w", filename, err)
	}

	err = json.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal state file %s: %w", filename, err)
	}

	sm.logger.Debug("State loaded", zap.String("file", filename), zap.String("path", path))
	return nil
}

// DeleteStateFile removes a state file
func (sm *StateManager) DeleteStateFile(filename string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	path := sm.GetStatePath(filename)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state file %s: %w", filename, err)
	}

	sm.logger.Debug("State file deleted", zap.String("file", filename), zap.String("path", path))
	return nil
}

// ListStateFiles returns a list of all state files
func (sm *StateManager) ListStateFiles() ([]string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	files, err := os.ReadDir(sm.stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	var filenames []string
	for _, file := range files {
		if !file.IsDir() {
			filenames = append(filenames, file.Name())
		}
	}

	return filenames, nil
}

// ClearAllState removes all state files
func (sm *StateManager) ClearAllState() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	files, err := os.ReadDir(sm.stateDir)
	if err != nil {
		return fmt.Errorf("failed to read state directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			path := filepath.Join(sm.stateDir, file.Name())
			err = os.Remove(path)
			if err != nil {
				sm.logger.Warn("Failed to delete state file", zap.String("file", file.Name()), zap.Error(err))
			}
		}
	}

	sm.logger.Info("All state files cleared")
	return nil
}
