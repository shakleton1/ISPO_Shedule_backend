package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRefreshToken_Success(t *testing.T) {
	token, err := GenerateRefreshToken()
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// Base64 URL-safe, 32 bytes => 43 characters
	assert.Len(t, token, 43)
}

func TestGenerateRefreshToken_MultipleTokens(t *testing.T) {
	token1, err := GenerateRefreshToken()
	require.NoError(t, err)
	
	token2, err := GenerateRefreshToken()
	require.NoError(t, err)
	
	// Токены должны быть уникальными
	assert.NotEqual(t, token1, token2)
}

func TestHashRefreshToken_Success(t *testing.T) {
	raw := "test-refresh-token"
	
	hash, err := HashRefreshToken(raw)
	
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	// SHA256 => 64 hex characters
	assert.Len(t, hash, 64)
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	raw := "same-token"
	
	hash1, err := HashRefreshToken(raw)
	require.NoError(t, err)
	
	hash2, err := HashRefreshToken(raw)
	require.NoError(t, err)
	
	// Хэши должны быть одинаковыми
	assert.Equal(t, hash1, hash2)
}

func TestHashRefreshToken_EmptyToken(t *testing.T) {
	hash, err := HashRefreshToken("")
	
	assert.Error(t, err)
	assert.Empty(t, hash)
	assert.Contains(t, err.Error(), "refresh_token required")
}

func TestHashRefreshToken_DifferentTokens(t *testing.T) {
	hash1, err := HashRefreshToken("token1")
	require.NoError(t, err)
	
	hash2, err := HashRefreshToken("token2")
	require.NoError(t, err)
	
	assert.NotEqual(t, hash1, hash2)
}
