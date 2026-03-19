//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

func TestRepositoryRefreshTokens_CreateAndGetByHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	user := &auth.User{
		Login:        fmt.Sprintf("rt_user_%d", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         auth.RoleStudent,
	}
	require.NoError(t, db.Create(user).Error)
	defer db.Where("id = ?", user.ID).Delete(&auth.User{})

	tokenHash := fmt.Sprintf("hash-%d", time.Now().UnixNano())
	expiresAt := time.Now().UTC().Add(2 * time.Hour)

	created, err := repo.CreateRefreshToken(user.ID, tokenHash, expiresAt)
	require.NoError(t, err)
	defer db.Where("id = ?", created.ID).Delete(&auth.RefreshToken{})

	assert.Equal(t, user.ID, created.UserID)
	assert.Equal(t, tokenHash, created.TokenHash)

	got, err := repo.GetRefreshTokenByHash(tokenHash)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, user.ID, got.UserID)
}

func TestRepositoryRefreshTokens_RevokeRefreshToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	user := &auth.User{
		Login:        fmt.Sprintf("rt_revoke_user_%d", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         auth.RoleStudent,
	}
	require.NoError(t, db.Create(user).Error)
	defer db.Where("id = ?", user.ID).Delete(&auth.User{})

	baseHash := fmt.Sprintf("base-hash-%d", time.Now().UnixNano())
	replacementHash := fmt.Sprintf("replacement-hash-%d", time.Now().UnixNano())

	base, err := repo.CreateRefreshToken(user.ID, baseHash, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	replacement, err := repo.CreateRefreshToken(user.ID, replacementHash, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	defer db.Where("id IN ?", []int64{base.ID, replacement.ID}).Delete(&auth.RefreshToken{})

	require.NoError(t, repo.RevokeRefreshToken(base.ID, &replacement.ID))

	var row auth.RefreshToken
	require.NoError(t, db.First(&row, base.ID).Error)
	require.NotNil(t, row.RevokedAt)
	require.NotNil(t, row.ReplacedByTokenID)
	assert.Equal(t, replacement.ID, *row.ReplacedByTokenID)
}

func TestRepositoryRefreshTokens_RevokeAllForUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	userA := &auth.User{
		Login:        fmt.Sprintf("rt_all_a_%d", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         auth.RoleStudent,
	}
	userB := &auth.User{
		Login:        fmt.Sprintf("rt_all_b_%d", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         auth.RoleStudent,
	}
	require.NoError(t, db.Create(userA).Error)
	require.NoError(t, db.Create(userB).Error)
	defer db.Where("id IN ?", []int64{userA.ID, userB.ID}).Delete(&auth.User{})

	tA1, err := repo.CreateRefreshToken(userA.ID, fmt.Sprintf("a1-%d", time.Now().UnixNano()), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	tA2, err := repo.CreateRefreshToken(userA.ID, fmt.Sprintf("a2-%d", time.Now().UnixNano()), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	tB1, err := repo.CreateRefreshToken(userB.ID, fmt.Sprintf("b1-%d", time.Now().UnixNano()), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	defer db.Where("id IN ?", []int64{tA1.ID, tA2.ID, tB1.ID}).Delete(&auth.RefreshToken{})

	require.NoError(t, repo.RevokeAllRefreshTokensForUser(userA.ID))

	var rows []auth.RefreshToken
	require.NoError(t, db.Where("id IN ?", []int64{tA1.ID, tA2.ID, tB1.ID}).Find(&rows).Error)
	require.Len(t, rows, 3)

	byID := map[int64]auth.RefreshToken{}
	for _, r := range rows {
		byID[r.ID] = r
	}

	require.NotNil(t, byID[tA1.ID].RevokedAt)
	require.NotNil(t, byID[tA2.ID].RevokedAt)
	assert.Nil(t, byID[tB1.ID].RevokedAt)
}
