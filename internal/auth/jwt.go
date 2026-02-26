package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	Role    Role  `json:"role"`
	GroupID *int  `json:"group_id,omitempty"`
	Subgroup *int16 `json:"subgroup,omitempty"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if secret == "" {
		return nil, fmt.Errorf("jwt_secret is empty")
	}
	return &TokenManager{secret: []byte(secret), ttl: ttl}, nil
}

func (m *TokenManager) IssueAccessToken(u *User, now time.Time) (string, time.Time, error) {
	exp := now.Add(m.ttl)
	claims := Claims{
		Role:     u.Role,
		GroupID:  u.GroupID,
		Subgroup: u.Subgroup,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", u.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return s, exp, nil
}

func (m *TokenManager) Parse(tokenString string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
