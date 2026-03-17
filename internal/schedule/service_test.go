package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Integration tests для Service с реальной БД находятся в
// internal/integration/auth_middleware_test.go с build tag integration

func TestNewService_Success(t *testing.T) {
	repo := &Repository{}
	deps := ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	}
	
	svc := NewService(deps)
	
	require.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC), svc.semesterStartDate)
	assert.NotNil(t, svc.weekCache)
}

func TestService_GetCurrentWeek_InvalidGroupID(t *testing.T) {
	repo := &Repository{}
	svc := NewService(ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})
	
	resp, err := svc.GetCurrentWeek(0, time.Now())
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "group_id required")
}

func TestService_GetRange_InvalidGroupID(t *testing.T) {
	repo := &Repository{}
	svc := NewService(ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})
	
	resp, err := svc.GetRange(0, time.Now(), time.Now())
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "group_id required")
}

func TestService_GetRange_EndDateBeforeStartDate(t *testing.T) {
	repo := &Repository{}
	svc := NewService(ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})
	
	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	
	resp, err := svc.GetRange(1, startDate, endDate)
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "date_end before date_start")
}
