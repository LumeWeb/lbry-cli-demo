package shared

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// AccountManager handles account creation and login operations
type AccountManager struct {
	client       *LBRYPortalClient
	logger       *zap.Logger
	stateManager *StateManager
}

// NewAccountManager creates a new account manager
func NewAccountManager(client *LBRYPortalClient, logger *zap.Logger) (*AccountManager, error) {
	stateManager, err := NewStateManager(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &AccountManager{
		client:       client,
		logger:       logger,
		stateManager: stateManager,
	}, nil
}

// CreateAndLoginAccount creates a new account and logs in
func (am *AccountManager) CreateAndLoginAccount() (*FakeAccountData, error) {
	// Create fake account
	fakeAccount := GenerateFakeAccount()

	am.logger.Info("Creating account",
		zap.String("email", fakeAccount.Email),
		zap.String("first_name", fakeAccount.FirstName),
		zap.String("last_name", fakeAccount.LastName))

	// Register user
	err := am.client.RegisterUser(fakeAccount.Email, fakeAccount.Password, fakeAccount.FirstName, fakeAccount.LastName)
	if err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	am.logger.Info("Account created successfully, logging in...")

	// Login
	err = am.client.Login(fakeAccount.Email, fakeAccount.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Save account state for reuse
	err = am.stateManager.SaveAccountState(fakeAccount)
	if err != nil {
		am.logger.Warn("Failed to save account state", zap.Error(err))
	}

	am.logger.Info("Login successful!")
	return fakeAccount, nil
}

// LoginWithCredentials logs in with existing credentials
func (am *AccountManager) LoginWithCredentials(email, password string) error {
	am.logger.Info("Logging in with existing credentials", zap.String("email", email))

	err := am.client.Login(email, password)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	am.logger.Info("Login successful!")
	return nil
}

// LoginOrCreateAccount attempts to login with existing account state, or creates a new one
func (am *AccountManager) LoginOrCreateAccount() (*FakeAccountData, error) {
	// Try to load existing account state
	accountState, err := am.stateManager.LoadAccountState()
	if err != nil {
		return nil, fmt.Errorf("failed to load account state: %w", err)
	}

	if accountState != nil {
		// Found existing account, try to login
		am.logger.Info("Found existing account, attempting login", zap.String("email", accountState.Email))

		err = am.LoginWithCredentials(accountState.Email, accountState.Password)
		if err != nil {
			am.logger.Warn("Failed to login with existing account, creating new one", zap.Error(err))
			return am.CreateAndLoginAccount()
		}

		// Convert AccountState back to FakeAccountData
		fakeAccount := &FakeAccountData{
			Email:     accountState.Email,
			Password:  accountState.Password,
			FirstName: accountState.FirstName,
			LastName:  accountState.LastName,
		}

		am.logger.Info("Successfully logged in with existing account")
		return fakeAccount, nil
	}

	// No existing account, create a new one
	am.logger.Info("No existing account found, creating new one")
	return am.CreateAndLoginAccount()
}

// CleanupAllStreams unpins/deletes all streams for the current account
func (am *AccountManager) CleanupAllStreams() error {
	am.logger.Info("Cleaning up all streams for the account")

	// List all streams
	streams, err := am.ListAndLogStreams()
	if err != nil {
		return fmt.Errorf("failed to list streams for cleanup: %w", err)
	}

	if len(streams) == 0 {
		am.logger.Info("No streams to cleanup")
		return nil
	}

	// Delete each stream
	for i, stream := range streams {
		am.logger.Info("Deleting stream",
			zap.Int("index", i),
			zap.String("sd_hash", stream.SDHash),
			zap.String("stream_hash", stream.StreamHash))

		err = am.client.DeleteStream(stream.SDHash)
		if err != nil {
			// Check if it's a 404 error (stream already deleted)
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Stream not found") {
				am.logger.Info("Stream already deleted or not found",
					zap.String("sd_hash", stream.SDHash))
			} else {
				am.logger.Warn("Failed to delete stream",
					zap.String("sd_hash", stream.SDHash),
					zap.Error(err))
			}
		} else {
			am.logger.Info("Successfully deleted stream", zap.String("sd_hash", stream.SDHash))
		}
	}

	am.logger.Info("Stream cleanup completed", zap.Int("total_streams", len(streams)))
	return nil
}

// WaitForOperations waits for all account operations to complete
func (am *AccountManager) WaitForOperations() error {
	am.logger.Info("Waiting for all operations to complete...")

	err := am.client.WaitForAllOperations()
	if err != nil {
		return fmt.Errorf("failed while waiting for operations: %w", err)
	}

	am.logger.Info("All operations completed successfully!")
	return nil
}

// ListAndLogStreams lists streams and logs their information
func (am *AccountManager) ListAndLogStreams() ([]StreamResponse, error) {
	am.logger.Info("Listing streams...")

	streams, err := am.client.ListStreams()
	if err != nil {
		return nil, fmt.Errorf("failed to list streams: %w", err)
	}

	am.logger.Info("Found streams", zap.Int("count", len(streams.Data)))
	for i, stream := range streams.Data {
		am.logger.Info("Stream",
			zap.Int("index", i),
			zap.Int("id", stream.ID),
			zap.String("sd_hash", stream.SDHash),
			zap.String("stream_hash", stream.StreamHash))
	}

	return streams.Data, nil
}
