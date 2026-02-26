package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func handlePublicListGroups(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListGroups()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handlePublicListSubjects(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListSubjects()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handlePublicListLocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListLocations()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}
