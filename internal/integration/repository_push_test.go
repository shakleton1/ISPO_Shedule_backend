//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/schedule"
)

func TestRepositoryPush_UpsertAndDeleteDeviceToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group1 := &schedule.Group{Name: fmt.Sprintf("Push Group A %d", time.Now().UnixNano()), Course: 1}
	require.NoError(t, db.Create(group1).Error)
	defer db.Where("id = ?", group1.ID).Delete(&schedule.Group{})

	group2 := &schedule.Group{Name: fmt.Sprintf("Push Group B %d", time.Now().UnixNano()), Course: 2}
	require.NoError(t, db.Create(group2).Error)
	defer db.Where("id = ?", group2.ID).Delete(&schedule.Group{})

	token := fmt.Sprintf("push-token-%d", time.Now().UnixNano())

	require.NoError(t, repo.UpsertDeviceToken(group1.ID, token))
	require.NoError(t, repo.UpsertDeviceToken(group2.ID, token))

	var row schedule.DeviceToken
	require.NoError(t, db.Where("token = ?", token).First(&row).Error)
	assert.Equal(t, group2.ID, row.GroupID)

	require.NoError(t, repo.DeleteDeviceToken(token))

	var count int64
	require.NoError(t, db.Model(&schedule.DeviceToken{}).Where("token = ?", token).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestRepositoryPush_ListDeviceTokensByGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := &schedule.Group{Name: fmt.Sprintf("Push List Group %d", time.Now().UnixNano()), Course: 1}
	require.NoError(t, db.Create(group).Error)
	defer db.Where("id = ?", group.ID).Delete(&schedule.Group{})

	tokenA := fmt.Sprintf("token-a-%d", time.Now().UnixNano())
	tokenB := fmt.Sprintf("token-b-%d", time.Now().UnixNano())
	require.NoError(t, repo.UpsertDeviceToken(group.ID, tokenA))
	require.NoError(t, repo.UpsertDeviceToken(group.ID, tokenB))
	defer db.Where("token IN ?", []string{tokenA, tokenB}).Delete(&schedule.DeviceToken{})

	rows, err := repo.ListDeviceTokensByGroup(group.ID)
	require.NoError(t, err)

	got := map[string]bool{}
	for _, r := range rows {
		got[r.Token] = true
	}
	assert.True(t, got[tokenA])
	assert.True(t, got[tokenB])
}
