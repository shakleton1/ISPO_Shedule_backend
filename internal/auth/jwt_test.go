package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenManager_EmptySecret(t *testing.T) {
	tm, err := NewTokenManager("", time.Hour)
	
	assert.Nil(t, tm)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jwt_secret is empty")
}

func TestNewTokenManager_Success(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	
	require.NoError(t, err)
	require.NotNil(t, tm)
	assert.Equal(t, time.Hour, tm.ttl)
}

func TestTokenManager_IssueAccessToken_Success(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	user := &User{
		ID:           1,
		Login:        "testuser",
		Role:         RoleAdmin,
		GroupID:      nil,
		Subgroup:     nil,
		PasswordHash: "hash",
	}
	
	now := time.Now().UTC()
	token, exp, err := tm.IssueAccessToken(user, now)
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.WithinDuration(t, now.Add(time.Hour), exp, time.Second)
}

func TestTokenManager_IssueAccessToken_DifferentRoles(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	roles := []Role{RoleAdmin, RoleDispatcher, RoleViewer, RoleStudent}
	
	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			user := &User{ID: 1, Role: role}
			token, exp, err := tm.IssueAccessToken(user, time.Now())
			
			require.NoError(t, err)
			assert.NotEmpty(t, token)
			assert.WithinDuration(t, time.Now().Add(time.Hour), exp, time.Second)
			
			claims, parseErr := tm.Parse(token)
			require.NoError(t, parseErr)
			assert.Equal(t, role, claims.Role)
		})
	}
}

func TestTokenManager_IssueAccessToken_WithGroupAndSubgroup(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	groupID := 5
	subgroup := int16(1)
	user := &User{
		ID:       1,
		Role:     RoleStudent,
		GroupID:  &groupID,
		Subgroup: &subgroup,
	}
	
	token, exp, err := tm.IssueAccessToken(user, time.Now())
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(time.Hour), exp, time.Second)
	
	claims, parseErr := tm.Parse(token)
	require.NoError(t, parseErr)
	assert.Equal(t, RoleStudent, claims.Role)
	require.NotNil(t, claims.GroupID)
	assert.Equal(t, groupID, *claims.GroupID)
	require.NotNil(t, claims.Subgroup)
	assert.Equal(t, subgroup, *claims.Subgroup)
}

func TestTokenManager_Parse_ValidToken(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	user := &User{
		ID:       42,
		Login:    "testuser",
		Role:     RoleDispatcher,
		GroupID:  nil,
		Subgroup: nil,
	}
	
	token, _, err := tm.IssueAccessToken(user, time.Now())
	require.NoError(t, err)
	
	claims, err := tm.Parse(token)
	
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "42", claims.Subject)
	assert.Equal(t, RoleDispatcher, claims.Role)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
}

func TestTokenManager_Parse_ExpiredToken(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	user := &User{ID: 1, Role: RoleAdmin}
	
	past := time.Now().Add(-2 * time.Hour)
	token, _, err := tm.IssueAccessToken(user, past)
	require.NoError(t, err)
	
	claims, err := tm.Parse(token)
	
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenManager_Parse_InvalidSignature(t *testing.T) {
	tm1, err := NewTokenManager("secret1", time.Hour)
	require.NoError(t, err)
	
	tm2, err := NewTokenManager("secret2", time.Hour)
	require.NoError(t, err)
	
	user := &User{ID: 1, Role: RoleAdmin}
	token, _, err := tm1.IssueAccessToken(user, time.Now())
	require.NoError(t, err)
	
	claims, err := tm2.Parse(token)
	
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenManager_Parse_WrongSigningMethod(t *testing.T) {
	// Создаём токен с неправильным методом подписи
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	// Парсим токен, созданный с другим методом (это должно завершиться ошибкой)
	// Для этого создадим ручной токен с HS384
	claims := Claims{
		Role: RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	// Создаём токен с HS384 вместо HS256
	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	wrongTokenString, err := wrongToken.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	parsedClaims, err := tm.Parse(wrongTokenString)

	assert.Error(t, err)
	assert.Nil(t, parsedClaims)
}

func TestTokenManager_Parse_InvalidTokenString(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	claims, err := tm.Parse("invalid.token.string")
	
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenManager_Parse_MalformedToken(t *testing.T) {
	tm, err := NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)
	
	tests := []string{
		"",
		"onlyonepart",
		"invalid..",
		"eyJhbGciOiJIUzI1NiJ9", // только header
	}
	
	for _, token := range tests {
		t.Run(token, func(t *testing.T) {
			claims, err := tm.Parse(token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}
