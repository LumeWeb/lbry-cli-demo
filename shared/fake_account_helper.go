package shared

import "github.com/brianvoe/gofakeit/v7"

// FakeAccountData holds generated fake account data
type FakeAccountData struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
}

// GenerateFakeAccount creates fake account data for testing/registration
func GenerateFakeAccount() *FakeAccountData {
	return &FakeAccountData{
		FirstName: gofakeit.FirstName(),
		LastName:  gofakeit.LastName(),
		Email:     gofakeit.Email(),
		Password:  gofakeit.Password(true, true, true, true, false, 12),
	}
}
