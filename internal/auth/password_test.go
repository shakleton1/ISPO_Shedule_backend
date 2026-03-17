package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_VerifyPassword_Success(t *testing.T) {
	password := "testPassword123!"

	hash, err := HashPassword(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, VerifyPassword(hash, password))
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "correctPassword123"
	wrongPassword := "wrongPassword456"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	assert.False(t, VerifyPassword(hash, wrongPassword))
}

func TestHashPassword_Deterministic(t *testing.T) {
	password := "samePassword123"

	hash1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, err := HashPassword(password)
	require.NoError(t, err)

	// Хэши должны быть разными (bcrypt использует salt)
	assert.NotEqual(t, hash1, hash2)

	// Но оба должны верифицироваться с правильным паролем
	assert.True(t, VerifyPassword(hash1, password))
	assert.True(t, VerifyPassword(hash2, password))
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, VerifyPassword(hash, ""))
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	assert.False(t, VerifyPassword("invalid_hash", "password"))
	assert.False(t, VerifyPassword("", "password"))
}
