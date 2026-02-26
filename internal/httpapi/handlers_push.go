package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if err := repo.UpsertDeviceToken(req.GroupID, req.Token); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusCreated)
	}
}

func handlePushUnregister(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req pushUnregisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if err := repo.DeleteDeviceToken(req.Token); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
