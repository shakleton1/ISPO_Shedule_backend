package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

// Exported handlers for integration tests

// HandlePushRegisterForTest exports handlePushRegister for integration tests
func HandlePushRegisterForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handlePushRegister(repo)
}

// HandlePushUnregisterForTest exports handlePushUnregister for integration tests
func HandlePushUnregisterForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handlePushUnregister(repo)
}

type pushRegisterRequest struct {
	GroupID int    `json:"group_id"`
	Token   string `json:"token"`
}

type pushUnregisterRequest struct {
	Token string `json:"token"`
}

func handlePushRegister(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req pushRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.GroupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}
		if req.Token == "" {
			writeValidationError(c, "token", "token required")
			return
		}
		if err := repo.UpsertDeviceToken(req.GroupID, req.Token); err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "registered", "group_id": req.GroupID})
	}
}

func handlePushUnregister(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req pushUnregisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.Token == "" {
			writeValidationError(c, "token", "token required")
			return
		}
		if err := repo.DeleteDeviceToken(req.Token); err != nil {
			writeDBError(c, err)
			return
		}
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}
